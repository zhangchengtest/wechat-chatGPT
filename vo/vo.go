package vo

import "time"

type ArticleResultVO struct {
	Code      int32
	Data      *Article
	Message   string
	isSuccess bool
}

type NovelResultVO struct {
	Code      int32
	Data      *Novel
	Message   string
	isSuccess bool
}

type Novel struct {
	Content string
	Url     string `json:"url"`
}

type Article struct {
	Id        int64
	Chapter   int32
	Title     string
	Content   string
	ReadCount int32
	Category  string
	UpdateBy  string     `json:"updateBy"`
	UpdateDt  *time.Time `json:"updateDt"`
	CreateDt  time.Time  `json:"createDt"`
	CreateBy  string     `json:"createBy"`
}
