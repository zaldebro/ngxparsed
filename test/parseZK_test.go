package parse_zk

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreparation (t *testing.T) {
	gitBasePath := "./ngxGit/conf"

	ClusterDir, err := ioutil.ReadDir(gitBasePath)
	if err != nil {
		fmt.Println("读取目录失败：", err)
	}

	for _, clusterConfig := range ClusterDir {
		// 如果不是目录，即为相关配置文件，nginx.conf，proxy.conf 等
		if !clusterConfig.IsDir() {
			continue
		}

		clusterConfPath := filepath.Join(gitBasePath, clusterConfig.Name())
		HandleClusterConfig(clusterConfPath)
	}
}


func HandleClusterConfig (path string) {
	clusterConf, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println("读取集群配置文件失败：", err)
		return
	}

	for _, conf := range clusterConf {
		if conf.Name() == "site-enable" && conf.IsDir() {
			fmt.Println("move site-enable to target confDir")
		}
		if conf.Name() == "upstream_zk_nodes.conf" {

			zkPath := filepath.Join(path, conf.Name())

			fi, err := os.Open(zkPath)
			if err != nil {
				fmt.Println("读取文件异常：", err)
				return
			}

			r := bufio.NewReader(fi)

			preGlMap := make(map[string]string)

			for {
				line, err := r.ReadString('\n')
				line = strings.TrimSpace(line)

				if strings.HasPrefix(line, "#") {
					continue
				}

				args := strings.Split(line, ":")

				if len(args) > 1 && args[1] != "" { // zk_node: 会被识别成长度为 2
					preGlMap[args[0]] = strings.Join(args[1:], ":")
				}

				if err != nil && err != io.EOF {
					fmt.Println("读取文件异常：", err)
				}

				if err == io.EOF {
					break
				}
			}
			fmt.Println("preGlMap--> ", preGlMap)
		}
	}

}



func TestSplit (t *testing.T) {
	line := "zk_no:"
	args := strings.Split(line, ":")
	fmt.Println(len(args))

	if args[1] == "" {
		fmt.Println("a?")
	}

}

