package server

import (
	"strings"
	"testing"

	"github.com/pschlafley/coding-challenges/go-memcache/types"
)

var (
	setData     = "set test 0 30 4"
	getData     = "get test"
	addData     = "add test 0 30 4"
	replaceData = "replace test 0 30 4"
)

func dataParser(cmd *types.ServerCmd, data []byte) *types.ServerCmd {
	dataSlice := strings.Split(string(data), " ")

	switch dataSlice[0] != "" {
	case dataSlice[0] != "set" && dataSlice[0] != "get" && dataSlice[0] != "add" && dataSlice[0] != "replace":
		cmd.DataBlock = string(data)
	case dataSlice[0] == "set":
		cmd.Command = string(data)
		cmd.DataBlock = ""
	case dataSlice[0] == "get":
		cmd.Command = string(data)
	case dataSlice[0] == "add":
		cmd.Command = string(data)
	case dataSlice[0] == "replace":
		cmd.Command = string(data)
	}

	return cmd
}

func TestDataParserSet(t *testing.T) {
	cmd := &types.ServerCmd{}

	s := dataParser(cmd, []byte(setData))

	if s.Command != setData {
		t.Fatalf("expected= %s, got= %s", setData, s.Command)
	}
}

func TestDataParserGet(t *testing.T) {
	cmd := &types.ServerCmd{}

	s := dataParser(cmd, []byte(getData))

	if s.Command != getData {
		t.Fatalf("expected= %s, got= %s", getData, s.Command)
	}
}
func TestDataParserAdd(t *testing.T) {
	cmd := &types.ServerCmd{}

	s := dataParser(cmd, []byte(addData))

	if s.Command != addData {
		t.Fatalf("expected= %s, got= %s", addData, s.Command)
	}
}
func TestDataParserReplace(t *testing.T) {
	cmd := &types.ServerCmd{}

	s := dataParser(cmd, []byte(replaceData))

	if s.Command != replaceData {
		t.Fatalf("expected= %s, got= %s", replaceData, s.Command)
	}
}

func deleteNodeFromQueue() types.Queue[int] {
	q := types.Queue[int]{}

	q.Enque(1)
	q.Enque(2)
	q.Enque(3)
	q.Enque(4)
	q.Enque(5)
	q.Enque(6)

	q.Deque()

	return q
}

func TestQueueDelete(t *testing.T) {
	cMap := []int{}
	tMap := []int{}

	test_queue := deleteNodeFromQueue()

	control_queue := types.Queue[int]{}
	control_queue.Enque(1)
	control_queue.Enque(2)
	control_queue.Enque(3)
	control_queue.Enque(4)
	control_queue.Enque(5)

	cnode := control_queue.Head()
	tnode := test_queue.Head()

	for cnode != nil {
		cMap = append(cMap, cnode.Value())
		cnode = cnode.Next()
	}

	for tnode != nil {
		tMap = append(tMap, tnode.Value())
		tnode = tnode.Next()
	}

	for i := 0; i < len(tMap) && i < len(cMap); i++ {
		if tMap[i] != cMap[i] {
			t.Fatalf("expected: %v, got: %v", cMap, tMap)
		}

	}
}
