package shared

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"

	"math/rand"
)

var ElectionCompletedSignal = make(chan bool)

var ItemList = []string{"salt", "fish", "boar"}
var LeaderID = 1
var PostList = make(map[string][]string)

// ============================== Struct ==============================
type Node struct {
	ID         int         // Node unique ID
	Address    string      // Node's network address
	Clock      int         // Logical clock for synchronization
	Role       interface{} // Node's role: Seller, Buyer, or Coordinator
	Mutex      sync.Mutex  // Mutex for thread safety
	IsActive   bool        // Whether the node is active
	NeighborID []int
	Post       int
	Trader     string
}

// Seller represents a seller in the system
type Seller struct {
	Post         string         // Post number for the seller
	ItemsForSale map[string]int // Items and their quantities
}

// Buyer represents a buyer in the system
type Buyer struct {
	Post         string         // Post number for the buyer
	ShoppingList map[string]int // Items the buyer wants to purchase
}

// Coordinator represents a trader responsible for managing peers
type Trader struct {
	Peer        string // Id of the trader peer
	Post        string
	HeartbeatMu sync.Mutex // for heartbeat synchronization
	Heartbeat   bool
}

type Warehouse struct {
	Items map[string]int // Item name -> Quantity
	// Locks      map[string]*sync.Mutex // Locks for each product
	GlobalLock sync.Mutex // Global lock for synchronizing inventory access
	Mutex      sync.Mutex
}

// Message is used to represent messages during the trading process
type Message struct {
	From     int
	To       int
	Type     string // Request, Success, Failure, Pay
	Item     string
	Quantity int
	Message  string
}

// Args and Reply structures for the RPC method, DisplayNodes
type DisplayNodesRequest struct{}

type DisplayNodesResponse struct {
	Message string
	Details string
}

// ============================== Methods ==============================
/*
StartServer starts the RPC server for the node
*/
func (n *Node) StartServer() error {
	server := rpc.NewServer()
	n.Address = fmt.Sprintf("localhost:800%d", n.ID)
	serviceName := fmt.Sprintf("Node%d", n.ID)
	err := server.RegisterName(serviceName, n)
	if err != nil {
		return fmt.Errorf("failed to register RPC server for Node %d: %v", n.ID, err)
	}

	listener, err := net.Listen("tcp", n.Address)
	if err != nil {
		return fmt.Errorf("error starting server on %s: %v", n.Address, err)
	}
	log.Printf("Node %d is listening on %s", n.ID, n.Address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection on Node %d: %v", n.ID, err)
			continue
		}
		go server.ServeConn(conn)
	}

}

/*
AssignRole assigns a role (Seller, Buyer, or Coordinator) to a Node
*/
func (n *Node) AssignRole(role string, postNum string) {
	n.Mutex.Lock()
	defer n.Mutex.Unlock()

	item := ItemList[rand.Intn(len(ItemList))]
	quantity := rand.Intn(10) + 1

	switch role {
	case "seller":
		n.Role = &Seller{
			Post:         postNum,
			ItemsForSale: make(map[string]int),
		}
		n.Role.(*Seller).ItemsForSale[item] = quantity

	case "buyer":
		n.Role = &Buyer{
			Post:         postNum,
			ShoppingList: make(map[string]int),
		}
		n.Role.(*Buyer).ShoppingList[item] = quantity

	case "trader":
		n.Role = &Trader{
			Post: postNum,
		}
		n.Trader = Int2String(n.ID)
	default:
		fmt.Printf("Invalid role: %s\n", role)
	}
}

/*
GetClient returns an RPC client to communicate with another node
*/
func GetClient(toID int) (*rpc.Client, error) {
	address := fmt.Sprintf("localhost:%d", 8000+toID)

	for retries := 0; retries < 3; retries++ {
		client, err := rpc.Dial("tcp", address)
		if err == nil {
			return client, nil
		}
		log.Printf("Retrying connection to Node %d (%d/3): %v", toID, retries+1, err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to Node %d after 3 retries", toID)
}

/*
DisplayNodes displays the information of the node
*/
func (n *Node) DisplayNodes(req *DisplayNodesRequest, res *DisplayNodesResponse) error {

	var details string
	fmt.Printf("\n[Node ID: %d Role Information]\n", n.ID)
	switch role := n.Role.(type) {
	case *Seller:
		details = fmt.Sprintf("Node ID: %d, Role: seller , Post:%s, Leader: %s,  Neighbors: %v, ItemsforSale: %v",
			n.ID, role.Post, n.Trader, n.NeighborID, role.ItemsForSale)
	case *Buyer:
		details = fmt.Sprintf("Node ID: %d, Role: buyer, Post:%s, Leader: %s, Neighbors: %v, ShoppingList: %v",
			n.ID, role.Post, n.Trader, n.NeighborID, role.ShoppingList)
	case *Trader:
		details = fmt.Sprintf("Node ID: %d, Role: Trader, Post:%s, Leader: %s,  Neighbors: %v,",
			n.ID, role.Post, n.Trader, n.NeighborID)
	default:
		details = fmt.Sprintf("Node ID: %d, Leader: %s,   Neighbors: %v",
			n.ID, n.Trader, n.NeighborID)
	}

	// Populate the response
	res.Message = "Node information retrieved successfully."
	res.Details = details

	return nil
}

/*
Terminate terminates the node
*/
func (n *Node) Terminate(req *Message, res *Message) error {
	log.Printf("Node %d: Terminating...", n.ID)

	// Perform cleanup here if needed
	res.Type = "SUCCESS"
	res.Message = fmt.Sprintf("Node %d terminated.", n.ID)

	go func() {
		time.Sleep(1 * time.Second) // Give time for response to be sent
		os.Exit(0)
	}()

	return nil
}

/*
Returns the role of the node
*/
func (n *Node) IsSeller() bool {
	return n.GetRole() == "seller"
}

func (n *Node) IsBuyer() bool {
	return n.GetRole() == "buyer"
}

func (n *Node) IsTrader() bool {
	return n.GetRole() == "trader"
}

func (n *Node) GetRole() string {
	switch n.Role.(type) {
	case *Seller:
		return "seller"
	case *Buyer:
		return "buyer"
	case *Trader:
		return "trader"
	default:
		return "unrecognized or not assigned"
	}
}

func (n *Node) GetTrader(req *Message, reply *Message) error {
	// Ensure the node is part of the correct post
	if req.Type != "GET_TRADER" {
		return fmt.Errorf("invalid request type: %s", req.Type)
	}

	// Return the trader for the node's post
	reply.Type = "SUCCESS"
	reply.Message = n.Trader // Trader ID for the node's post
	return nil
}

// ============================== Helper Functions ==============================
/*
String2Int converts a string to an integer
*/
func String2Int(value string) (int, error) {
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid string for conversion: %q", value)
	}
	return intValue, nil
}

/*
Int2String converts an integer to a string
*/
func Int2String(value int) string {
	stringValue := fmt.Sprint(value)
	return stringValue
}
