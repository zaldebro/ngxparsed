package test

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"testing"
	"time"
)

var DB *gorm.DB

var dsn = "root:123456@tcp(192.168.179.131:3306)/domain?charset=utf8mb4&parseTime=True"

func init() {
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

	err = DB.Migrator().AutoMigrate(&User{}, &Blog{},)
	if err != nil {
		log.Fatal("创建数据库失败：", err)
	}
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

type BaseUser struct {
	gorm.Model
	UserName string `json:"user_name" gorm:"comment:用户名"`
}

func (BaseUser) TableName() string {
	return "base_user"
}

type User struct {
	BaseUser
	Blogs []Blog `json:"blogs" gorm:"foreignKey:UserId;references:ID"`
}

func (User) TableName() string {
	return "user"
}

type Blog struct {
	gorm.Model
	Title   string `json:"title" gorm:"comment:标题"`
	Content string `json:"content" gorm:"comment:正文"`
	UserId  uint   `json:"user_id" gorm:"comment:作者ID"`
}

func TestWriteRecord(t *testing.T) {
	var baseUser = BaseUser{
		UserName: "zhangsan",
	}

	var blogs []Blog

	blog1 := Blog{
		Title:   "title1",
		Content: "content1",
		//UserId:  12,
	}

	blog2 := Blog{
		Title:   "title2",
		Content: "content2",
		//UserId:  13,
	}

	blogs = append(blogs, blog1)
	blogs = append(blogs, blog2)

	var user = User{
		BaseUser: baseUser,
		Blogs:    blogs,
	}

	res := DB.Create(&user)

	fmt.Println("res: ", res)
}