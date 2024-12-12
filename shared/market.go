package shared

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

// TraderIDRequest represents a request to get the Trader ID for a specific post.
type TraderIDRequest struct {
	Post int
}

// TraderIDResponse represents the response containing the Trader ID.
type TraderIDResponse struct {
	TraderID string
	Message  string
}

// GetTraderID is an RPC method that returns the Trader ID for the specified post.
func (n *Node) GetTraderID(req *TraderIDRequest, res *TraderIDResponse) error {
	if n.Post != req.Post {
		res.TraderID = ""
		res.Message = fmt.Sprintf("Node %d does not manage Post %d.", n.ID, req.Post)
		return nil
	}

	if n.Trader == "" {
		res.TraderID = ""
		res.Message = fmt.Sprintf("Post %d has no Trader assigned.", req.Post)
		return nil
	}

	res.TraderID = n.Trader
	res.Message = fmt.Sprintf("Trader for Post %d is Node %s.", req.Post, n.Trader)
	return nil
}

/**
 * UpdateInventory updates the inventory in the warehouse node based on the request type (LOAD or BUY).
 */
func (w *Node) UpdateInventory(req *Message, reply *Message) error {
	warehouse := w.Role.(*Warehouse)
	warehouse.GlobalLock.Lock()         // Acquire global lock
	defer warehouse.GlobalLock.Unlock() // Release global lock

	// Load the inventory from file before processing
	w.LoadInventoryFromFile("inventory.txt")

	switch req.Type {
	case "LOAD":
		w.Role.(*Warehouse).Items[req.Item] += req.Quantity
		reply.Type = "SUCCESS"
		reply.Message = fmt.Sprintf("Added %d units of %s to inventory", req.Quantity, req.Item)
		log.Printf("SUCCESS - Warehouse: Added %d units of %s. Current inventory: %v", req.Quantity, req.Item, w.Role.(*Warehouse).Items)

	case "BUY":
		if w.Role.(*Warehouse).Items[req.Item] >= req.Quantity {
			w.Role.(*Warehouse).Items[req.Item] -= req.Quantity
			reply.Type = "SUCCESS"
			reply.Message = fmt.Sprintf("Shipped %d units of %s", req.Quantity, req.Item)
			log.Printf("SUCCESS - Warehouse: Shipped %d units of %s. Current inventory: %v", req.Quantity, req.Item, w.Role.(*Warehouse).Items)
		} else {
			reply.Type = "FAILURE"
			reply.Message = fmt.Sprintf("Not enough %s in inventory. Available: %d", req.Item, w.Role.(*Warehouse).Items[req.Item])
			log.Printf("REJECTED - Warehouse: Insufficient %s in inventory. Available: %d", req.Item, w.Role.(*Warehouse).Items[req.Item])
		}

	default:
		reply.Type = "FAILURE"
		reply.Message = "Unknown request type"
		log.Printf("Warehouse: ERROR - Unknown request type: %s", req.Type)
	}

	// Save the updated inventory to file
	err := w.SaveInventoryToFile("inventory.txt")
	if err != nil {
		reply.Type = "FAILURE"
		reply.Message = "Failed to save inventory to file"
		log.Printf("ERROR - Warehouse: Error saving inventory: %v", err)
		return err
	}

	return nil
}

/*
saveInventoryToFile saves the warehouse's inventory to a file.
*/
func (w *Node) SaveInventoryToFile(filename string) error {
	w.Role.(*Warehouse).Mutex.Lock()         // Acquire local lock for file operations
	defer w.Role.(*Warehouse).Mutex.Unlock() // Release local lock

	file, err := os.Create(filename)
	if err != nil {
		log.Printf("ERROR - Warehouse: Error opening inventory file for writing: %v", err)
		return err
	}
	defer file.Close()

	for item, quantity := range w.Role.(*Warehouse).Items {
		_, err := fmt.Fprintf(file, "%s %d\n", item, quantity)
		if err != nil {
			log.Printf(" ERROR - Warehouse: error writing inventory data: %v", err)
			return err
		}
	}
	return nil
}

/*
LoadInventoryFromFile loads the warehouse's inventory from a file.
*/
func (n *Node) LoadInventoryFromFile(filename string) {
	if w, ok := n.Role.(*Warehouse); ok {
		w.Mutex.Lock()
		defer w.Mutex.Unlock()

		file, err := os.Open(filename)
		if err != nil {
			log.Printf("ERROR - error opening inventory file: %v", err)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var item string
			var quantity int
			fmt.Sscanf(scanner.Text(), "%s %d", &item, &quantity)
			w.Items[item] = quantity
		}
		log.Printf("SUCCESS - Warehouse: Loaded inventory from file: %v", w.Items)
	}
}
