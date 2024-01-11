package types

type Node[T any] struct {
	value T
	next  *Node[T]
	prev  *Node[T]
}

func (n *Node[T]) Value() T {
	return n.value
}

func (n *Node[T]) Next() *Node[T] {
	return n.next
}

type Queue[T any] struct {
	head *Node[T]
	tail *Node[T]
}

func (q *Queue[T]) Enque(item T) {
	var newNode *Node[T] = &Node[T]{value: item}

	if q.head == nil {
		q.head = newNode
		q.tail = newNode
	} else {
		oldNode := q.head
		oldNode.prev = newNode
		newNode.next = oldNode
		q.head = newNode
	}
}

func (q *Queue[T]) Delete(node *Node[T]) *Queue[T] {
	if node.prev == nil || node.next == nil {
		node = nil
	} else {
		prevNode := node.prev
		nextNode := node.next

		prevNode.next = node.next
		nextNode.prev = node.prev

		node.next = nil
		node.prev = nil
		node = nil
	}

	return q
}

func (q *Queue[T]) Head() *Node[T] {
	return q.head
}

func (q *Queue[T]) Tail() *Node[T] {
	return q.tail
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{}
}

type ServerCmd struct {
	Command   string
	DataBlock string
}

type DataArgs struct {
	Key       string
	DataBlock string
	Flags     int
	Exptime   int64
	ByteCt    int
	Noreply   bool
}

type Store struct {
	Db *map[string]*DataArgs
}
