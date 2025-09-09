// Package tree  尝试归一树形结构
// byd
package tree

type Record interface {
	NextLayer() []Record
	GetId() uint
	GetName() string
}
type BottomRecord interface {
	Record
	GetScore(string, int64, int64) float64
}
type Node struct {
	Id       uint    `json:"id"`
	Name     string  `json:"name"`
	Score    float64 `json:"score"`
	Children []*Node `json:"children"`
}

func Unfold3[A, P Record, C BottomRecord](a *A, userId string, startTime, endTime int64) *Node {
	n := Node{
		Id:    (*a).GetId(),
		Name:  (*a).GetName(),
		Score: 0.0,
	}

	for _, p := range (*a).NextLayer() {
		pp, ok := (p).(P)
		if ok {
			panic("类型错误")
		}
		nn := Unfold2[P, C](&pp, userId, startTime, endTime)
		n.Children = append(n.Children, nn)
		n.Score += nn.Score
	}
	return &n
}
func Unfold2[P Record, C BottomRecord](p *P, userId string, startTime, endTime int64) *Node {
	n := Node{
		Id:    (*p).GetId(),
		Name:  (*p).GetName(),
		Score: 0.0,
	}

	for _, c := range (*p).NextLayer() {
		cc, ok := (c).(C)
		if !ok {
			panic("类型错误")
		}
		nn := Node{
			Id:    cc.GetId(),
			Name:  cc.GetName(),
			Score: cc.GetScore(userId, startTime, endTime),
		}
		n.Children = append(n.Children, &nn)
		n.Score += nn.Score
	}
	return &n
}
func ToRecordSlice[T Record](list []T) []Record {
	result := make([]Record, len(list))
	for i := range list {
		result[i] = list[i]
	}
	return result
}
