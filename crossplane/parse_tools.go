package crossplane

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"regexp"
	"strings"
	"time"
)

// AssistParsing 二次处理解析结构体
type AssistParsing struct {
	glVar         map[string]string   // 存放 http 域变量
	GlUpstream    map[string][]string // 存放 http 域 upstream
	ParsePipeline chan ParDomain
	MYDBPipeline  chan DBDomain
}

// DisposeConFile 解析 nginx 主配置文件，主要是解析出 全局变量和后端
func (a *AssistParsing) DisposeConFile (httpDirectives *Directive) {
	// 如果 nginx.conf 配置文件写入了 server，则需要等全局处理完成后在处理
	var serverSlice []*Directive
	for _, httpDirect := range httpDirectives.Block {
		// 收集 http 域变量，貌似 http 域不能使用 set
		if httpDirect.Directive == "set" {
			fmt.Println(httpDirect.Args)
			a.glVar[httpDirect.Args[0]] = httpDirect.Args[1]
		}

		// 收集 http 域 upstream map[upstreamName][]backends
		if httpDirect.Directive == "upstream" {
			//a.glUpstream[httpDirect.Args[0]] = httpDirect.Args
			name := httpDirect.Args[0]
			for _, backend := range httpDirect.Block {
				a.GlUpstream[name] = append(a.GlUpstream[httpDirect.Args[0]], backend.Args[0])
			}
		}
		// 存储 server block
		if httpDirect.Directive == "server" {
			serverSlice = append(serverSlice, httpDirect)
		}
	}

	// 如果 nginx.conf 中写入了 server 块，在这里解析
	if len(serverSlice) > 0 {
		a.DisposeServerBlock(serverSlice, a.GlUpstream, httpDirectives.File)
	}
}

// DisposeImpConFile 解析被导入的 nginx 配置文件
func (a *AssistParsing) DisposeImpConFile (serverDirectives Config) {

	impUpstream := make(map[string][]string)
	// 需要等待 upstream block 解析完成才能够解析 server block
	var serverSlice []*Directive
	for _, serverDirect := range serverDirectives.Parsed {
		// 存储 upstream
		if serverDirect.Directive == "include" {
			fmt.Println("include block: ", serverDirect)
		}

		if serverDirect.Directive == "upstream" {
			name := serverDirect.Args[0]
			for _, backend := range serverDirect.Block {
				if backend.Directive == "server" {
					impUpstream[name] = append(impUpstream[name], backend.Args[0])
				}
			}
		}
		// 存储server block
		if serverDirect.Directive == "server" {
			serverSlice = append(serverSlice, serverDirect)
		}
	}
	if len(serverSlice) > 0 {
		a.DisposeServerBlock(serverSlice, impUpstream, serverDirectives.File)
	}
}


type ParDomain struct {
	FilePath string
	Domains []string
	Ports []string
	Locations []ParLocation
}

type ParLocation struct {
	//Rule string
	Path []string // 写法不规范： = / {...}   =/ {...}
	backend string
	backends []string
}

// DisposeServerBlock 解析 server block
func (a *AssistParsing) DisposeServerBlock (serverSlice []*Directive, Upstreams map[string][]string, filepath string) {
	// 深拷贝 http 域的变量
	varMap := a.DeepCopyMap(a.glVar)
	for _, serverBlock := range serverSlice {
		var parDomain ParDomain
		parDomain.FilePath = filepath

		for _, block := range serverBlock.Block {

			// 收集监听端口
			if block.Directive == "listen" {
				parDomain.Ports = append(parDomain.Ports, block.Args...)
			}

			// 存储 server_name
			if block.Directive == "server_name" {
				parDomain.Domains = append(parDomain.Domains, block.Args...)
			}

			// 存储变量
			if block.Directive == "set" {
				// set $name ${val}val 写法不合法，不考虑
				arg := a.DisposeVar(varMap, block.Args[1])
				varMap[block.Args[0]] = arg
			}

			// 存储 location 信息
			if block.Directive == "location" {
				var parLocation ParLocation
				parLocation.Path = block.Args
				loVarMap := a.DeepCopyMap(varMap)

				for _, loBlock := range block.Block {
					if loBlock.Directive == "set" {

						arg := a.DisposeVar(loVarMap, loBlock.Args[1])
						loVarMap[loBlock.Args[0]] = arg

					}

					if loBlock.Directive == "proxy_pass" || loBlock.Directive == "fastcgi_pass" {
						arg := loBlock.Args[0]
						// 去掉变量的 http:// 或者 https:// 前缀
						arg = strings.TrimPrefix(arg, "http://")
						arg = strings.TrimPrefix(arg, "https://")
						// 去掉路径
						arg = strings.Split(arg, "/")[0]
						// 解析变量
						arg = a.DisposeVar(loVarMap,arg)
						// 存储 upstream 的名字
						parLocation.backend = arg
						if backend, ok := Upstreams[arg]; ok {
							parLocation.backends = backend
						} else {
							parLocation.backends = loBlock.Args
						}
					}
				}

				// 收集 location 信息
				parDomain.Locations = append(parDomain.Locations, parLocation)
			}
		}
		//fmt.Println("parDomain--> ", parDomain)
		a.ParsePipeline <- parDomain
	}
}

// DeepCopyMap 深拷贝 map
func (a *AssistParsing) DeepCopyMap (loMap map[string]string) (deepMap map[string]string){
	deepMap = make(map[string]string, len(loMap))
	for k, v := range loMap{
		deepMap[k] = v
	}
	return deepMap
}


// DisposeVar 替换变量
func (a *AssistParsing) DisposeVar (loVar VarStruct, arg string) string {

	//  定义正则表达式，匹配 ${name}和 $value
	re := regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)
	//  提取占位符
	matches := re.FindAllStringSubmatch(arg, -1)

	// 如果没有匹配到变量，则直接返回
	if len(matches) == 0 {
		return arg
	}
	fmt.Println("matches--> ", matches)
	for _, match := range matches {
		// match[1]是${name}中的name，match[2]是$value中的value
		if match[1] != "" {  // set $name value  替换 map[$name] -> ${name}
			if varValue, ok := loVar["$" + match[1]]; ok {
				arg = strings.Replace(arg, match[0], varValue, -1)
			}
		} else {  // map[$name] -> $name
			if varValue, ok := loVar[match[0]]; ok {
				arg = strings.Replace(arg, match[0], varValue, -1)
			}
		}
	}

	return arg
}

type DBDomain struct {
	gorm.Model
	FilePath  string        `gorm:"type:varchar(255)"`
	Domain   string        `gorm:"type:text"` // 存储为JSON
	Port     string        `gorm:"type:text"` // 存储为JSON
	Locations []DBLocation `gorm:"foreignKey:DomainID;references:ID"` // 一对多关系
}

func (DBDomain) TableName() string {
	return "domain"
}

type DBLocation struct {
	gorm.Model
	DomainID    uint
	Path        string   `gorm:"type:text"` // 存储为JSON
	BackendName string   `gorm:"type:text"`
	Backends    []DBServer `gorm:"foreignKey:LocationID;references:ID"` // 一对多关系
}

func (DBLocation) TableName() string {
	return "location"
}

type DBServer struct {
	gorm.Model
	LocationID uint
	Server    string `gorm:"type:varchar(255)"`
}

func (DBServer) TableName() string {
	return "server"
}

func (a *AssistParsing) PipelineToMYDB () {
	for parDomain := range a.ParsePipeline {

		// 将解析到的数据转换成 gorm，每个域名一个数据
		for _, domaiName := range parDomain.Domains {

			// location
			var dbLocations []DBLocation
			// gorm 需要每个数据是单独的变量，否则会导致 id 雷同
			for _, location := range parDomain.Locations {
				var dbLocation DBLocation
				dbLocation.Path = strings.Join(location.Path, "")
				dbLocation.BackendName = location.backend

				for _, backendServer := range location.backends {
					var dbServer DBServer
					dbServer.Server = backendServer
					dbLocation.Backends = append(dbLocation.Backends, dbServer)
				}
				dbLocations = append(dbLocations, dbLocation)
			}

			// domain
			var dbDomain DBDomain
			dbDomain.FilePath = parDomain.FilePath
			dbDomain.Domain = domaiName
			dbDomain.Port = parDomain.Ports[0]
			dbDomain.Locations = dbLocations

			a.MYDBPipeline <- dbDomain
		}
	}

	// 当没有数据需要处理时，关闭数据库 chan
	close(a.MYDBPipeline)
}


func setPool(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(10)
}


func (a *AssistParsing) WtireToMYDB () {

	var DB *gorm.DB
	var dsn = "root:123456@tcp(192.168.179.131:3306)/domain?charset=utf8mb4"
	var err error
	DB, err = gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
		DefaultStringSize: 256,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		PrepareStmt: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	setPool(DB)

	err = DB.Migrator().AutoMigrate(&DBDomain{}, &DBLocation{}, &DBServer{})

	if err != nil {
		log.Fatal("创建数据库失败：", err)
	}

	for dbDomain := range a.MYDBPipeline {
		//fmt.Println("domain: ", dbDomain)
		res := DB.Create(&dbDomain)
		if res.Error != nil {
			fmt.Println("创建 dbDomain 失败：", res.Error)
			return
		}

		//fmt.Println("res: ", *res)
	}
}


