package vo

import "time"

type ResultVO struct {
	Code      int32
	Data      *Article
	Message   string
	isSuccess bool
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
