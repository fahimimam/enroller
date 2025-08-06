package model

var (
	Models []interface{}
)

func init() {
	Models = append(Models, &UserInfo{})
}
