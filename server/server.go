package server

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pschlafley/coding-challenges/go-memcache/types"
)

type Server struct {
	ListenAddr string
	Listener   net.Listener
	quit       chan struct{}
	MsgCh      chan types.Message
	PeerMap    map[net.Addr]string
	Store      *types.Store
}

func NewServer(address string) *Server {
	dbMap := make(map[string]*types.DataArgs)

	store := &types.Store{
		Db:   &dbMap,
		Size: 5,
	}

	return &Server{
		ListenAddr: address,
		quit:       make(chan struct{}),
		MsgCh:      make(chan types.Message),
		PeerMap:    make(map[net.Addr]string),
		Store:      store,
	}
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
	fmt.Println(s.Store.Size)

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

	setMsgStruct := &types.Message{}
	getMsgStruct := &types.Message{}

	switch parsedCmd[0] != "" {
	case parsedCmd[0] == "set" && cmd.DataBlock != "":
		if len((*s.Store.Db)) > 5 {
			setMsgStruct.Cmd = types.ServerCmd{}
			setMsgStruct.RemoteAddr = conn.RemoteAddr()
			setMsgStruct.Text = "Store is at it's maximum capacity!"
			s.MsgCh <- *setMsgStruct
			return
		}

		setMsgStruct.Cmd = types.ServerCmd{
			Command:   cmd.Command,
			DataBlock: cmd.DataBlock,
		}
		setMsgStruct.RemoteAddr = conn.RemoteAddr()

		cmdSlice := strings.Split(setMsgStruct.Cmd.Command, " ")

		setMsgStruct.Text = fmt.Sprintf("%s %s", cmdSlice[0], setMsgStruct.Cmd.DataBlock)
		setMsgStruct.TimeStamp = time.Now().Format(time.ANSIC)

		s.MsgCh <- *setMsgStruct

		result := handleSetData(*cmd, s.Store)
		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""

	case parsedCmd[0] == "get":
		result := handleGetData(parsedCmd, s.Store)

		if result != "END\r\n" {
			getMsgStruct.Cmd = types.ServerCmd{
				Command:   cmd.Command,
				DataBlock: cmd.DataBlock,
			}

			getMsgStruct.RemoteAddr = conn.RemoteAddr()

			cmdSlice := strings.Split(getMsgStruct.Cmd.Command, " ")
			resultSlice := strings.Split(strings.TrimSpace(result), "\n")

			getMsgStruct.Text = fmt.Sprintf("%s %s", cmdSlice[0], resultSlice[1])
			getMsgStruct.TimeStamp = time.Now().Format(time.ANSIC)

			s.MsgCh <- *getMsgStruct
		}

		conn.Write([]byte(result))

	case parsedCmd[0] == "add" && cmd.DataBlock != "":
		result := handleAddData(*cmd, s.Store)
		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""
	case parsedCmd[0] == "replace" && cmd.DataBlock != "":
		result := handleReplaceData(*cmd, s.Store)
		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""
	case parsedCmd[0] == "append" && cmd.DataBlock != "":
		result := handleAppendData(*cmd, s.Store)
		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""
	case parsedCmd[0] == "prepend" && cmd.DataBlock != "":
		result := handlePrependData(*cmd, s.Store)
		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""
	case parsedCmd[0] == "delete":
		result := handleDeleteData(*cmd, s.Store)
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

	for k := range *store.Db {
		if k == key {
			return "END\r\n"
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
			result := handleSetData(cmd, store)
			return result
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
			(*store.Db)[key].DataBlock = strings.TrimSpace(v.DataBlock) + strings.TrimSpace(cmd.DataBlock)
			result = "STORED\r\n"
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
			(*store.Db)[key].DataBlock = strings.TrimSpace(cmd.DataBlock) + strings.TrimSpace(v.DataBlock)
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
