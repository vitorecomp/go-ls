package models

import "time"

type File struct {
	Name     string
	Size     int64
	Modified time.Time
	Mode     string
	IsDir    bool
}
