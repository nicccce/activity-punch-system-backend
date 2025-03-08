package model

import "gorm.io/gen"

// 通过添加注释生成自定义方法

type Querier interface {
	// SELECT * FROM @@table WHERE id=@id
	GetByID(id int) (gen.T, error) // 返回结构体和error

	// GetByIDReturnMap 根据ID查询返回map
	// SELECT * FROM @@table WHERE id=@id
	GetByIDReturnMap(id int) (gen.M, error) // 返回 map 和 error

	// SELECT * FROM @@table WHERE studentID=@studentID
	GetUserByStudentID(studentID string) (*gen.T, error) // 返回数据和 error
}
