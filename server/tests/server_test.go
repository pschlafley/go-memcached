package server

import (
	"strings"
	"testing"

	"github.com/pschlafley/coding-challenges/go-memcache/types"
)

/* type DataArgs struct {
	Key       string
	DataBlock string
	Flags     int
	Exptime   int64
	ByteCt    int
	Noreply   bool
}*/

var (
	setData     = "set test 0 30 4"
	getData     = "get test"
	addData     = "add test 0 30 4"
	replaceData = "replace test 0 30 4"
	deleteData  = "delete test 0 0 5"
	dataArgsMap = map[string]types.DataArgs{
		"casey": {
			Key:       "casey",
			DataBlock: "VALUE 0 5 casey",
			Exptime:   0,
			ByteCt:    5,
			Noreply:   false,
		},
		"peyton": {
			Key:       "peyton",
			DataBlock: "VALUE 0 6 peyton",
			Exptime:   0,
			ByteCt:    6,
			Noreply:   false,
		},
		"andre": {
			Key:       "andre",
			DataBlock: "VALUE 0 5 andre",
			Exptime:   0,
			ByteCt:    5,
			Noreply:   false,
		},
		"jerika": {
			Key:       "jerika",
			DataBlock: "VALUE 0 6 andre",
			Exptime:   0,
			ByteCt:    6,
			Noreply:   false,
		},
	}
)

func dataParser(cmd *types.ServerCmd, data []byte) *types.ServerCmd {
	dataSlice := strings.Split(string(data), " ")

	switch dataSlice[0] != "" {
	case dataSlice[0] != "set" && dataSlice[0] != "get" && dataSlice[0] != "add" && dataSlice[0] != "replace" && dataSlice[0] != "delete":
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
	case dataSlice[0] == "delete":
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

func TestDataParserDelete(t *testing.T) {
	cmd := &types.ServerCmd{}

	s := dataParser(cmd, []byte(deleteData))

	if s.Command != deleteData {
		t.Fatalf("expected= %s, got= %s", deleteData, s.Command)
	}
}

func deleteKey(keyToDelete string, m map[string]types.DataArgs) string {
	result := ""

	for k := range m {
		if keyToDelete == k {
			delete(m, keyToDelete)
			result = keyToDelete
		}
	}
	return result
}

func TestDeleteFromMap(t *testing.T) {
	deletedKey := deleteKey("peyton", dataArgsMap)

	switch deletedKey != "" {
	case deletedKey == "casey":
		t.Logf("Success! Expected: %s, Got: %s", "casey", deletedKey)
	case deletedKey == "peyton":
		t.Logf("Success! Expected: %s, Got: %s", "peyton", deletedKey)
	case deletedKey == "andre":
		t.Logf("Success! Expected: %s, Got: %s", "andre", deletedKey)
	case deletedKey == "jerika":
		t.Logf("Success! Expected: %s, Got: %s", "jerika", deletedKey)
	default:
		t.Fatal("Test Failed, did not receive a deleted key")
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

// func TestOpenLogFile(t *testing.T) {
// 	file, err := os.Open("../logs/server.txt")

// 	defer file.Close()

// 	if err != nil {
// 		t.Fatal(err)
// 	} else {
// 		t.Log(file.Stat())
// 	}

// }
