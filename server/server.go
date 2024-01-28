package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pschlafley/coding-challenges/go-memcache/types"
)

type Server struct {
	ListenAddr string
	Listener   net.Listener
	quit       chan struct{}
	MsgCh      chan types.Message
	PeerMap    map[net.Addr]string
	mu         sync.Mutex
	Store      *types.Store
}

func NewServer(address string) *Server {
	dbMap := make(map[string]*types.DataArgs, 1)

	store := &types.Store{
		Db:   &dbMap,
		Size: 1000,
	}

	return &Server{
		ListenAddr: address,
		quit:       make(chan struct{}),
		MsgCh:      make(chan types.Message),
		PeerMap:    make(map[net.Addr]string),
		Store:      store,
	}
}

func OpenLogFile(fileName string) (*os.File, error) {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %s Error: %s", fileName, err)
	}

	return file, nil
}

func handleRemoveLogs(file *os.File) error {
	buf := bufio.NewScanner(file)

	for i := 0; i < 2; i++ {
		buf.Scan()
	}

	return nil
}

func (s *Server) HandleServerMessageQueue() {
	go func() {
		// since there could be multiple goroutines reading/writing from the messages queue and to the file
		// I am using a mutex to lock so that only one goroutine can perform a read/write at a time
		s.mu.Lock()
		// since we are doing it outside of the for loop we can safley defer the unlock method
		defer s.mu.Unlock()
		messageQueue := types.NewQueue[*types.Message]()
		for {
			file, err := OpenLogFile("./logs/server.log")

			if err != nil {
				log.Fatal(err)
			}

			msg := <-s.MsgCh

			messageQueue.Enque(&msg)

			node := messageQueue.Head()

			fmtString := fmt.Sprintf("%v %s: %s", node.Value().TimeStamp, node.Value().RemoteAddr, node.Value().Text)

			_, wErr := file.WriteString(strings.TrimSpace(fmtString))

			if wErr != nil {
				log.Fatal(wErr)
			}

			messageQueue.Deque()

			// handleRemoveLogs(file)

			file.Close()
		}
	}()
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)

	if err != nil {
		return err
	}

	defer ln.Close()

	s.Listener = ln

	go s.AcceptConnections()

	// Wait here for the quit channel until that is done, if the quit channel is done then we can defer the ln.Close() func and clean everything up
	<-s.quit

	// If we are stopping our server other people can still be reading from the MsgCh so we need to notify them that we are closing the channel by closing it
	close(s.MsgCh)

	return nil
}

func (s *Server) AcceptConnections() {
	var i int = 1

	for {
		conn, err := s.Listener.Accept()

		if err != nil {
			fmt.Println("accept error: ", err)
			continue
		}

		fmt.Println("New Connection: ", conn.RemoteAddr())

		peerVal := fmt.Sprintf("conn%d", i)

		s.PeerMap[conn.RemoteAddr()] = peerVal

		i += 1

		// Each time we accept a connection, we will spin up a new goroutine so that it is not blocking and handle each connection in it's own goroutine
		go s.ReadConnections(conn)
	}
}

func (s *Server) ReadConnections(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 2048)

	cmd := &types.ServerCmd{}

	for {
		n, err := conn.Read(buf)

		if err != nil {
			fmt.Printf("connection closed: %s\n", conn.RemoteAddr())
			delete(s.PeerMap, conn.RemoteAddr())
			return
		}

		data := buf[:n]

		s.dataParser(conn, cmd, data)
	}
}

func (s *Server) dataParser(conn net.Conn, cmd *types.ServerCmd, data []byte) {
	dataSlice := strings.Split(string(data), " ")

	switch dataSlice[0] != "" {
	case dataSlice[0] != "set" && dataSlice[0] != "get" && dataSlice[0] != "add" && dataSlice[0] != "replace" && dataSlice[0] != "append" && dataSlice[0] != "prepend" && dataSlice[0] != "delete" && dataSlice[0] != "increment" && dataSlice[0] != "decrement":
		cmd.DataBlock = string(data)
	case dataSlice[0] == "set":
		cmd.Command = string(data)
		cmd.DataBlock = ""
	case dataSlice[0] == "get":
		cmd.Command = string(data)
	case dataSlice[0] == "add":
		cmd.Command = string(data)
		cmd.DataBlock = ""
	case dataSlice[0] == "replace":
		cmd.Command = string(data)
		cmd.DataBlock = ""
	case dataSlice[0] == "append":
		cmd.Command = string(data)
		cmd.DataBlock = ""
	case dataSlice[0] == "prepend":
		cmd.Command = string(data)
		cmd.DataBlock = ""
	case dataSlice[0] == "delete":
		cmd.Command = string(data)
	case dataSlice[0] == "increment":
		cmd.Command = string(data)
		cmd.DataBlock = ""
	case dataSlice[0] == "decrement":
		cmd.Command = string(data)
		cmd.DataBlock = ""
	}

	s.commandParser(cmd, conn)
}

func (s *Server) commandParser(cmd *types.ServerCmd, conn net.Conn) {
	parsedCmd := strings.Split(cmd.Command, " ")

	msgStruct := &types.Message{}

	switch parsedCmd[0] != "" {
	case parsedCmd[0] == "set" && cmd.DataBlock != "":
		if len((*s.Store.Db)) > s.Store.Size {
			msgStruct.Cmd = types.ServerCmd{}
			msgStruct.RemoteAddr = conn.RemoteAddr()
			msgStruct.Text = "Store is at it's maximum capacity!\n"

			s.MsgCh <- *msgStruct
			conn.Write([]byte(msgStruct.Text))
			msgStruct.Cmd.Command = ""
			msgStruct.Cmd.DataBlock = ""

			var i int = 0
			for i < 1 {
				for k := range *s.Store.Db {
					delete(*s.Store.Db, k)
					i++
				}
			}

			i = 0
			break
		} else {
			msgStruct.Cmd = types.ServerCmd{
				Command:   cmd.Command,
				DataBlock: cmd.DataBlock,
			}
			msgStruct.RemoteAddr = conn.RemoteAddr()

			cmdSlice := strings.Split(msgStruct.Cmd.Command, " ")

			msgStruct.Text = fmt.Sprintf("%s %s %s %s %s\n", cmdSlice[0], cmdSlice[1], cmdSlice[2], cmdSlice[4], msgStruct.Cmd.DataBlock)
			msgStruct.TimeStamp = time.Now().Format(time.ANSIC)

			s.MsgCh <- *msgStruct
			msgStruct.Cmd.Command = ""
			msgStruct.Cmd.DataBlock = ""

			result := handleSetData(*cmd, s.Store)
			conn.Write([]byte(result))
			cmd.Command = ""
			cmd.DataBlock = ""
		}

	case parsedCmd[0] == "get":
		result := handleGetData(parsedCmd, s.Store)

		if strings.TrimSpace(result) != "END" {
			msgStruct.Cmd = types.ServerCmd{
				Command:   cmd.Command,
				DataBlock: cmd.DataBlock,
			}

			msgStruct.RemoteAddr = conn.RemoteAddr()

			cmdSlice := strings.Split(msgStruct.Cmd.Command, " ")
			resultSlice := strings.Split(strings.TrimSpace(result), "\n")

			msgStruct.Text = fmt.Sprintf("%s: %s %s\r", cmdSlice[0], resultSlice[0], resultSlice[1])
			msgStruct.TimeStamp = time.Now().Format(time.ANSIC)

			s.MsgCh <- *msgStruct
			msgStruct.Cmd.Command = ""
			msgStruct.Cmd.DataBlock = ""

		} else {
			msgStruct.Cmd = types.ServerCmd{
				Command:   cmd.Command,
				DataBlock: cmd.DataBlock,
			}

			msgStruct.RemoteAddr = conn.RemoteAddr()

			cmdSlice := strings.Split(msgStruct.Cmd.Command, " ")

			msgStruct.Text = fmt.Sprintf("%s: Failed! Key not found!\n", cmdSlice[0])
			msgStruct.TimeStamp = time.Now().Format(time.ANSIC)

			s.MsgCh <- *msgStruct
			msgStruct.Cmd.Command = ""
			msgStruct.Cmd.DataBlock = ""
		}

		conn.Write([]byte(result))

	case parsedCmd[0] == "add" && cmd.DataBlock != "":
		if len((*s.Store.Db)) > s.Store.Size {
			msgStruct.Cmd = types.ServerCmd{}
			msgStruct.RemoteAddr = conn.RemoteAddr()
			msgStruct.Text = "Store is at it's maximum capacity!"

			s.MsgCh <- *msgStruct
			conn.Write([]byte(msgStruct.Text))

			var i int = 0
			for i < 1 {
				for k := range *s.Store.Db {
					delete(*s.Store.Db, k)
					i++
				}
			}
			i = 0
			break
		} else {
			msgStruct.Cmd = types.ServerCmd{
				Command:   cmd.Command,
				DataBlock: cmd.DataBlock,
			}
			msgStruct.RemoteAddr = conn.RemoteAddr()
			cmdSlice := strings.Split(msgStruct.Cmd.Command, " ")
			msgStruct.TimeStamp = time.Now().Format(time.ANSIC)
			msgStruct.Text = fmt.Sprintf("%s %s %s %s %s\n", cmdSlice[0], cmdSlice[1], cmdSlice[2], cmdSlice[4], msgStruct.Cmd.DataBlock)

			result := handleSetData(*cmd, s.Store)

			if strings.TrimSpace(result) == "END" {
				msgStruct.Text = fmt.Sprintf("%s %s: Failed! The key %s already exists!\n", cmdSlice[0], strings.TrimSpace(msgStruct.Cmd.DataBlock), cmdSlice[1])
			}

			s.MsgCh <- *msgStruct

			conn.Write([]byte(result))
			msgStruct.Cmd.Command = ""
			msgStruct.Cmd.DataBlock = ""
			cmd.Command = ""
			cmd.DataBlock = ""
		}

	case parsedCmd[0] == "replace" && cmd.DataBlock != "":
		msgStruct.Cmd = types.ServerCmd{
			Command:   cmd.Command,
			DataBlock: cmd.DataBlock,
		}
		msgStruct.RemoteAddr = conn.RemoteAddr()
		cmdSlice := strings.Split(msgStruct.Cmd.Command, " ")
		msgStruct.TimeStamp = time.Now().Format(time.ANSIC)
		msgStruct.Text = fmt.Sprintf("%s %s %s %s %s\n", cmdSlice[0], cmdSlice[1], cmdSlice[2], cmdSlice[4], msgStruct.Cmd.DataBlock)

		result := handleReplaceData(*cmd, s.Store)

		if strings.TrimSpace(result) == "NOT_STORED" {
			msgStruct.Text = fmt.Sprintf("%s %s: Failed! Could not find that key!\n", cmdSlice[0], msgStruct.Cmd.DataBlock)
		}

		s.MsgCh <- *msgStruct

		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""

	case parsedCmd[0] == "append" && cmd.DataBlock != "":
		msgStruct.Cmd = types.ServerCmd{
			Command:   cmd.Command,
			DataBlock: cmd.DataBlock,
		}
		msgStruct.RemoteAddr = conn.RemoteAddr()
		cmdSlice := strings.Split(msgStruct.Cmd.Command, " ")
		msgStruct.TimeStamp = time.Now().Format(time.ANSIC)
		msgStruct.Text = fmt.Sprintf("%s %s %s %s %s\n", cmdSlice[0], cmdSlice[1], cmdSlice[2], cmdSlice[4], msgStruct.Cmd.DataBlock)

		result := handleAppendData(*cmd, s.Store)

		if strings.TrimSpace(result) == "NOT_STORED" {
			msgStruct.Text = fmt.Sprintf("%s %s: Failed! Could not find that key!\n", cmdSlice[0], msgStruct.Cmd.DataBlock)
		}

		s.MsgCh <- *msgStruct

		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""

	case parsedCmd[0] == "prepend" && cmd.DataBlock != "":
		msgStruct.Cmd = types.ServerCmd{
			Command:   cmd.Command,
			DataBlock: cmd.DataBlock,
		}
		msgStruct.RemoteAddr = conn.RemoteAddr()
		cmdSlice := strings.Split(msgStruct.Cmd.Command, " ")
		msgStruct.TimeStamp = time.Now().Format(time.ANSIC)
		msgStruct.Text = fmt.Sprintf("%s %s %s %s %s\n", cmdSlice[0], cmdSlice[1], cmdSlice[2], cmdSlice[4], msgStruct.Cmd.DataBlock)

		result := handlePrependData(*cmd, s.Store)

		if strings.TrimSpace(result) == "NOT_STORED" {
			msgStruct.Text = fmt.Sprintf("%s %s: Failed! Could not find that key!\n", cmdSlice[0], msgStruct.Cmd.DataBlock)
		}

		s.MsgCh <- *msgStruct

		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""

	case parsedCmd[0] == "delete":
		msgStruct.Cmd = types.ServerCmd{
			Command:   cmd.Command,
			DataBlock: cmd.DataBlock,
		}
		msgStruct.RemoteAddr = conn.RemoteAddr()
		cmdSlice := strings.Split(msgStruct.Cmd.Command, " ")
		msgStruct.TimeStamp = time.Now().Format(time.ANSIC)
		msgStruct.Text = fmt.Sprintf("%s %s\n", cmdSlice[0], cmdSlice[1])

		result := handleDeleteData(*cmd, s.Store)

		if strings.TrimSpace(result) == "END" {
			msgStruct.Text = fmt.Sprintf("%s %s: Failed! Could not find that key!\n", cmdSlice[0], msgStruct.Cmd.DataBlock)
		}

		s.MsgCh <- *msgStruct

		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""

	case parsedCmd[0] == "increment" && cmd.DataBlock != "":
		result := handleIncrementStoreSize(*cmd, s.Store)
		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""

	case parsedCmd[0] == "decrement" && cmd.DataBlock != "":
		result := handleDecrementStoreSize(*cmd, s.Store)
		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""
	}
}

func handleSetData(data types.ServerCmd, store *types.Store) string {
	cmdSlice := strings.Split(data.Command, " ")
	flags, fErr := strconv.Atoi(strings.TrimSpace(cmdSlice[2]))
	key := cmdSlice[1]

	if strings.TrimSpace(cmdSlice[0]) == "add" || strings.TrimSpace(cmdSlice[0]) == "set" {
		for k := range *store.Db {
			if k == key {
				return "END\r\n"
			}
		}
	}

	if fErr != nil {
		return "Error: Flags field is missing or not a valid number, please try again\r\n"
	}

	byteCt, bCtErr := strconv.Atoi(strings.TrimSpace(cmdSlice[4]))

	if bCtErr != nil {
		return "Error: Byte Count field is missing or not a valid number, please try again\r\n"
	}

	expTime, expErr := strconv.ParseInt(strings.TrimSpace(cmdSlice[3]), 0, 64)

	if expErr != nil {
		return "Error: Exptime field is missing or not a valid number, please try again\r\n"
	}

	var noreply bool

	if len(cmdSlice) < 6 {
		noreply = false
	} else if strings.TrimSpace(cmdSlice[5]) == "noreply" {
		noreply = true
	}

	var expirationTime int64

	if expTime == 0 {
		expirationTime = 0
	} else if expTime > 0 {
		expirationTime = time.Now().Unix() + expTime
	} else if expTime < 0 {
		expirationTime = -1
	}

	// handle if flags and byte are undefined
	dataArgs := &types.DataArgs{
		DataBlock: data.DataBlock,
		Flags:     flags,
		Exptime:   expirationTime,
		ByteCt:    byteCt,
		Noreply:   noreply,
	}

	(*store.Db)[key] = dataArgs

	if noreply {
		return ""
	}

	return "STORED\r\n"
}

func handleGetData(cmdString []string, store *types.Store) string {
	key := strings.TrimSpace(cmdString[1])
	var result string

	if len((*store.Db)) == 0 {
		return "END\r\n"
	} else if len((*store.Db)) > 0 {
		for k, v := range *store.Db {
			if key == strings.TrimSpace(k) {
				exp := v.Exptime

				if exp == 0 {
					result = fmt.Sprintf("VALUE %s %d %d\n%s\n", k, v.Flags, v.ByteCt, strings.TrimSpace(v.DataBlock))
					return result
				} else if time.Now().Unix() > exp || exp < 0 {
					delete(*store.Db, k)
					result = "END\r\n"
					return result
				}

				result = fmt.Sprintf("VALUE %s %d %d\n%s\n", k, v.Flags, v.ByteCt, strings.TrimSpace(v.DataBlock))
			} else if key == strings.TrimSpace(key) {
				result = "END\r\n"
			}
		}
	}

	return result
}

func handleAddData(cmd types.ServerCmd, store *types.Store) string {
	cmdSlice := strings.Split(cmd.Command, " ")
	key := cmdSlice[1]

	for k := range *store.Db {
		if k == "" {
			result := handleSetData(cmd, store)
			return result
		} else if key != k {
			result := handleSetData(cmd, store)
			return result
		} else if string(key) == k {
			return "NOT_STORED\r\n"
		}
	}

	return ""
}

func handleReplaceData(cmd types.ServerCmd, store *types.Store) string {
	cmdSlice := strings.Split(cmd.Command, " ")
	key := cmdSlice[1]

	for k := range *store.Db {
		if k == "" {
			result := handleSetData(cmd, store)
			return result
		} else if string(key) != k {
			return "NOT_STORED\r\n"
		} else if string(key) == k {
			flags, fErr := strconv.Atoi(strings.TrimSpace(cmdSlice[2]))

			if fErr != nil {
				return "Error: Flags field is missing or not a valid number, please try again\r\n"
			}

			byteCt, bCtErr := strconv.Atoi(strings.TrimSpace(cmdSlice[4]))

			if bCtErr != nil {
				return "Error: Byte Count field is missing or not a valid number, please try again\r\n"
			}

			expTime, expErr := strconv.ParseInt(strings.TrimSpace(cmdSlice[3]), 0, 64)

			if expErr != nil {
				return "Error: Exptime field is missing or not a valid number, please try again\r\n"
			}

			var noreply bool

			if len(cmdSlice) < 6 {
				noreply = false
			} else if strings.TrimSpace(cmdSlice[5]) == "noreply" {
				noreply = true
			}

			var expirationTime int64

			if expTime == 0 {
				expirationTime = 0
			} else if expTime > 0 {
				expirationTime = time.Now().Unix() + expTime
			} else if expTime < 0 {
				expirationTime = -1
			}

			// handle if flags and byte are undefined
			dataArgs := &types.DataArgs{
				DataBlock: cmd.DataBlock,
				Flags:     flags,
				Exptime:   expirationTime,
				ByteCt:    byteCt,
				Noreply:   noreply,
			}

			(*store.Db)[key] = dataArgs

			if noreply {
				return ""
			}

			return "STORED\r\n"
		}
	}

	return ""
}

func handleAppendData(cmd types.ServerCmd, store *types.Store) string {
	// add being able to add a space in the datablock string
	key := strings.TrimSpace(strings.Split(cmd.Command, " ")[1])
	var result string

	if len(*store.Db) == 0 {
		return "NOT_STORED\r\n"
	}

	for k, v := range *store.Db {
		if k == strings.TrimSpace(key) {
			(*store.Db)[key].DataBlock = strings.TrimSpace(v.DataBlock) + " " + strings.TrimSpace(cmd.DataBlock)
			result = "STORED\r\n"
			return result
		} else if k != key {
			result = "NOT_STORED\r\n"
			return result
		}
	}

	return result
}

func handlePrependData(cmd types.ServerCmd, store *types.Store) string {
	// add being able to add a space in the datablock string
	key := strings.TrimSpace(strings.Split(cmd.Command, " ")[1])
	var result string

	if len(*store.Db) == 0 {
		return "NOT_STORED\r\n"
	}

	for k, v := range *store.Db {
		if k == strings.TrimSpace(key) {
			(*store.Db)[key].DataBlock = strings.TrimSpace(cmd.DataBlock) + " " + strings.TrimSpace(v.DataBlock)
			result = "STORED\r\n"
		} else if k != key {
			result = "NOT_STORED\r\n"
			return result
		}
	}

	return result
}

func handleDeleteData(cmd types.ServerCmd, store *types.Store) string {
	cmdSlice := strings.Split(cmd.Command, " ")
	result := ""
	keyToDelete := cmdSlice[1]

	if len((*store.Db)) == 0 {
		return "END\r\n"
	}

	for k := range *store.Db {
		if strings.TrimSpace(keyToDelete) == k {
			delete(*store.Db, k)
			result = "DELETED\r\n"
			return result
		} else if keyToDelete != k {
			result = "END\r\n"
		}
	}

	return result
}

func handleIncrementStoreSize(cmd types.ServerCmd, store *types.Store) string {
	incrementValue, err := strconv.Atoi(cmd.DataBlock)

	if err != nil {
		return fmt.Sprintf("Error: %s\n", err)
	}

	store.Size += incrementValue

	return "INCREMENT\r\n"
}

func handleDecrementStoreSize(cmd types.ServerCmd, store *types.Store) string {
	decrementValue, err := strconv.Atoi(cmd.DataBlock)

	if err != nil {
		return fmt.Sprintf("Error: %s\n", err)
	}

	store.Size += decrementValue

	return "DECREMENT\r\n"
}
