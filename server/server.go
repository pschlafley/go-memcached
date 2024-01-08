package server

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pschlafley/coding-challenges/go-memcahce/types"
)

type Server struct {
	ListenAddr string
	Listener   net.Listener
	quit       chan struct{}
	MsgCh      chan []byte
	peerMap    map[net.Addr]string
	Store      *types.Store
}

func NewServer(address string) *Server {
	store := &types.Store{
		Db: types.NewQueue[types.DataArgs](),
	}

	return &Server{
		ListenAddr: address,
		quit:       make(chan struct{}),
		MsgCh:      make(chan []byte, 10),
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
	for {
		conn, err := s.Listener.Accept()

		if err != nil {
			fmt.Println("accept error: ", err)
			continue
		}

		fmt.Println("New Connection: ", conn.RemoteAddr())

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
			fmt.Printf("connection closed: %s", conn.RemoteAddr())
			return
		}

		data := buf[:n]

		s.handleCommand(conn, cmd, data)
	}
}

func (s *Server) handleCommand(conn net.Conn, cmd *types.ServerCmd, data []byte) {
	dataSlice := strings.Split(string(data), " ")

	switch dataSlice[0] != "" {
	case dataSlice[0] == "set":
		cmd.Command = string(data)
		cmd.DataBlock = ""
	case dataSlice[0] != "set" && dataSlice[0] != "get":
		cmd.DataBlock = string(data)
	case dataSlice[0] == "get":
		cmd.Command = string(data)
	}

	str := strings.Split(cmd.Command, " ")

	switch str[0] != "" {
	case str[0] == "set" && cmd.DataBlock != "":
		result := handleSetData(*cmd, s.Store)
		conn.Write([]byte(result))
		cmd.Command = ""
		cmd.DataBlock = ""
	case str[0] == "get":
		result := handleGetData(dataSlice, s.Store)
		conn.Write([]byte(result))
	case str[0] == "add":
		result := handleAddData(dataSlice, s.Store)
		conn.Write([]byte(result))
	case str[0] == "replace":
		result := handleReplaceData(dataSlice, s.Store)
		conn.Write([]byte(result))
	}

}

func handleSetData(data types.ServerCmd, store *types.Store) string {
	cmdSlice := strings.Split(data.Command, " ")
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
	dataArgs := types.DataArgs{
		Key:       cmdSlice[1],
		DataBlock: strings.TrimSpace(data.DataBlock),
		Flags:     flags,
		Exptime:   expirationTime,
		ByteCt:    byteCt,
		Noreply:   noreply,
	}

	store.Db.Enque(dataArgs)

	if noreply {
		return ""
	}

	return "STORED\r\n"
}

func handleGetData(cmdString []string, store *types.Store) string {
	key := strings.TrimSpace(cmdString[1])
	var result string

	node := store.Db.Head()

	for node != nil {
		if key != node.Value().Key {
			result = "END\r\n"
		} else if key == node.Value().Key {
			exp := node.Value().Exptime

			if exp == 0 {
				result = fmt.Sprintf("VALUE %s %d %d\r\n", node.Value().DataBlock, node.Value().Flags, node.Value().ByteCt)
				return result
			} else if time.Now().Unix() > exp || exp < 0 {
				result = "END\r\n"
				store.Db.Delete(node)
				return result
			}

			result = fmt.Sprintf("VALUE %s %d %d\r\n", node.Value().DataBlock, node.Value().Flags, node.Value().ByteCt)
			return result
		}
		node = node.Next()
	}

	return result
}

func handleAddData(cmdString []string, store *types.Store) string {
	return ""
}

func handleReplaceData(cmdString []string, store *types.Store) string {
	return ""
}
