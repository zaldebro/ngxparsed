# ngxparsed

解析自建 ngx 配置，输出 域名、location 和 需要解析变量的 backend【proxy_pass http://${arg1}str$arg2;】
主要代码在 parse_tools.go 中

type DBDomain struct {
	gorm.Model
	FilePath  string        `gorm:"type:varchar(255)"`
	Domain   string        `gorm:"type:text"`
	Port     string        `gorm:"type:text"`
	Locations []DBLocation `gorm:"foreignKey:DomainID;references:ID"`
}


type DBLocation struct {
	gorm.Model
	DomainID    uint
	Path        string   `gorm:"type:text"`
	BackendName string   `gorm:"type:text"`
	Backends    []DBServer `gorm:"foreignKey:LocationID;references:ID"`
}

type DBServer struct {
	gorm.Model
	LocationID uint
	Server    string `gorm:"type:varchar(255)"`
}
