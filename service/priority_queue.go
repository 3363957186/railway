package service

type Item struct {
	node          string
	allTime       int64
	transferTimes int64
	index         int
	price         float64
	specialTag    bool //如果上一条边由A到D且中转时间小于15分钟，则这个Tag为true
}

// PriorityQueue：最小堆的实现
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

// 按照最短时间排序，如果时间相同，则换乘次数少的优先
func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].allTime == pq[j].allTime {
		return pq[i].transferTimes < pq[j].transferTimes
	}
	return pq[i].allTime < pq[j].allTime
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index, pq[j].index = i, j
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*Item)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}

type Item2 struct {
	node          string
	allTime       int64
	transferTimes int64
	index         int
	price         float64
	specialTag    bool //如果上一条边由A到D且中转时间小于15分钟，则这个Tag为true
}

// PriorityQueue：最小堆的实现
type PriorityQueue2 []*Item2

func (pq PriorityQueue2) Len() int { return len(pq) }

// 按照最短时间排序，如果时间相同，则换乘次数少的优先
func (pq PriorityQueue2) Less(i, j int) bool {
	if pq[i].price == pq[j].price {
		return pq[i].transferTimes < pq[j].transferTimes
	}
	return pq[i].price < pq[j].price
}

func (pq PriorityQueue2) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index, pq[j].index = i, j
}

func (pq *PriorityQueue2) Push(x interface{}) {
	item := x.(*Item2)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue2) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}
