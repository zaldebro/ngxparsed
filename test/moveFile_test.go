package testFile

import (
	"fmt"
	git "github.com/go-git/go-git/v5"
	. "github.com/go-git/go-git/v5/_examples"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// TODO 根据 url 下载文件并重命名到指定位置
func TestDownloadByUrl (t *testing.T) {
	imgUrl := "https://git.n.xiaomi.com/mig2-sre/ngx-gameunion-matrix/-/archive/master/ngx-gameunion-matrix-master.zip"
	//imgUrl := "https://www.twle.cn/static/i/img1.jpg"

	// Get the data
	resp, err := http.Get(imgUrl)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	// 创建一个文件用于保存
	out, err := os.Create("ngx.git")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()

	// 然后将响应流和文件流对接起来
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
}


func TestPullGit (t *testing.T) {

	// 配置参数
	url := "https://git.n.xiaomi.com/fuqingfei/diskdetectionalarm.git"
	directory := "./ngxGit"
	token := "N8YUdspJp-Th3FBB7FxK" // git 仓库的 accesstoken

	// 删除目标目录，防止 git 拉取失败，报错：仓库已存在
	err := os.RemoveAll(directory)
	if err != nil {
		fmt.Println("删除目标目录失败: ", directory)
		return
	}

	r, err := git.PlainClone(directory, false, &git.CloneOptions{
		Auth: &gitHttp.BasicAuth{
			Username: "fuqingfei", // 用户名随意
			Password: token,
		},
		URL:      url,
		//Progress: os.Stdout, // 将拉取 git 信息输出到终端
	})
	CheckIfError(err)
	ref, err := r.Head()
	CheckIfError(err)
	commit, err := r.CommitObject(ref.Hash())
	CheckIfError(err)
	fmt.Println("Hash: ", commit.Hash)


	// 开始移动目录，/home/work/nginx/site-enable/
	dstDir := "./site-enable"
	err = os.RemoveAll(dstDir)
	if err != nil {
		fmt.Println("删除目标目录失败: ", dstDir)
		return
	}

	rooms, err := ioutil.ReadDir(directory)
	for _, room := range rooms {
		// 只处理 conf 目录
		if !room.IsDir() || room.Name() != "test" {
			continue
		}

		// 处理 test 目录
		//fmt.Println(room.Name())
		srcDir := filepath.Join(directory, room.Name())
		err := os.Rename(srcDir, dstDir)
		if err != nil {
			fmt.Println("移动目录出错:", err)
			return
		}
		fmt.Println("移动目录成功：", room.Name())
	}

}


// 将集群的配置文件的 site-enable 逐个移动到 /home/work... 中并进行解析
func TestShowDirs (t *testing.T) {

	// 清空
	pwd, _ := os.Getwd()
	fmt.Println(pwd)

	// 配置文件所在地 /home/work/nginx/site-enable
	dstBase := "../../conf/newConf/site-enable"

	// 当前仓库的地址
	roomDir := "../../conf/oldConf"
	rooms, err := ioutil.ReadDir(roomDir)
	if err != nil {
		fmt.Println("err: ", err)
	}

	for _, room := range rooms {
		if room.IsDir() {

			// 如果目标目录不为空，会导致移动失败；这里直接删除目标目录
			dir, _ := ioutil.ReadDir(dstBase)
			if len(dir) > 0 {
				err := os.RemoveAll(dstBase)
				if err != nil {
					fmt.Println("删除旧文件夹失败！")
				}
			}

			// 将相关配置文件移动到目的目录
			srcPath := filepath.Join(roomDir, room.Name() + "/site-enable")
			err := os.Rename(srcPath, dstBase)
			if err != nil {
				fmt.Println("移动目录出错:", err)
				return
			}

			// TODO 加上解析配置文件的逻辑即可

		}
	}
}












