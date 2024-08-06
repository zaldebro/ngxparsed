package crossplane

import (
	"regexp"
	"strings"
)


type MapStruct map[string]Directives
type VarStruct map[string]string

// DeepCopyMap 深拷贝map[string]string
func DeepCopyMap(loMap map[string]string) (deepMap map[string]string){
	deepMap = make(map[string]string)
	for k, v := range loMap{
		deepMap[k] = v
	}
	return deepMap
}


func DisposeVar(loVar VarStruct, arg string) string {

	//  定义正则表达式，匹配 ${name}和 $value
	re := regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)

	//  提取占位符
	matches := re.FindAllStringSubmatch(arg, -1)

	// 如果没有匹配到变量，则直接返回
	if len(matches) == 0 {
		//fmt.Println("noMatch")
		return arg
	}

	//// 去掉变量的 http:// 或者 https:// 前缀
	//arg = strings.TrimPrefix(arg, "http://")
	//arg = strings.TrimPrefix(arg, "https://")
	//// 去掉路径
	//arg = strings.Split(arg, "/")[0]

	for _, match := range matches {
		// match[1]是${name}中的name，match[2]是$value中的value
		if match[1] != "" {  // set $name value  替换 map[$name] -> ${name}
			if varValue, ok := loVar["$" + match[1]]; ok {
				arg = strings.Replace(arg, match[0], varValue, 1)
			}
		} else {  // map[$name] -> $name
			if varValue, ok := loVar[match[0]]; ok {
				arg = strings.Replace(arg, match[0], varValue, 1)
			}
		}
	}

	return arg
}


func DisposeServer (glBlock *Directive, upMap MapStruct, ansMap MapStruct, httpVar VarStruct){
	serVar := DeepCopyMap(httpVar)

	//fmt.Println("upMap---> ", upMap)

	// 存储当前 server 层的变量
	for _, v := range glBlock.Block{

		//if v.Directive == "server_name"{
		//	fmt.Println()
		//	fmt.Println("server_name: ", v.Args[0])
		//}

		if v.Directive == "set"{
			serVar[v.Args[0]] = DisposeVar(serVar, v.Args[1])
		}
	}

	// 开始解析 location
	for _, loBlock := range glBlock.Block{
		if loBlock.Directive == "location" && len(loBlock.Block) > 0 {
			// 存储当前 location 层的变量
			loVar := DeepCopyMap(serVar)
			proxySlice := [][]string{}

			for _, v := range loBlock.Block{
				if v.Directive == "set"{
					loVar[v.Args[0]] = DisposeVar(loVar, v.Args[1])
				}
				if v.Directive == "proxy_pass" {
					proxySlice = append(proxySlice, v.Args)
				}
			}

			// 对当前 location 遍历完成后，开始替换变量
			for _, proxy := range proxySlice {
				arg := proxy[0]
				// 去掉变量的 http:// 或者 https:// 前缀
				arg = strings.TrimPrefix(arg, "http://")
				arg = strings.TrimPrefix(arg, "https://")
				// 去掉路径
				arg = strings.Split(arg, "/")[0]
				tmpProxy := DisposeVar(loVar, arg)
				//tmpProxy := DisposeVar(loVar, proxy[0])
				// 对当前的 proxy 变量替换完成，尝试替换后端的 upstream
				// 如果在当前的 upstream map 里面，则替换
				// 否则，保留
				if backend, ok := upMap[tmpProxy];ok{
					//fmt.Println("proxy:", loBlock.Args[0])
					//fmt.Println("backend:", backend)
					ansMap[loBlock.Args[0]] = backend
				} else {

					var pared Directives
					ans := &Directive{
						Directive: tmpProxy,
						Line:      1,
						Args:      []string{},
						File:      "",
					}
					pared = append(pared, ans)
					ansMap[loBlock.Args[0]] = pared
					//fmt.Println("proxy:", loBlock.Args[0])
					//fmt.Println("backend:", pared)
				}
			}
		}
	}

}