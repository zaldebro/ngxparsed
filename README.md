# ngxparsed

解析自建 ngx 配置，输出 域名、location 和 需要解析变量的 backend【proxy_pass http://${arg1}str$arg2;】
主要代码在 parse_tools.go 中
