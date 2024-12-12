package shared

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

/*
 * StartBuyer starts the buyer node to send buy requests to the trader.
 */
func (buyer *Node) StartBuyer(args *Message, reply *Message) error {
	if _, ok := buyer.Role.(*Buyer); ok {
		// Start sending buy requests every 10 seconds
		go func() {
			ticker := time.NewTicker(10 * time.Second) // Adjust interval as needed
			defer ticker.Stop()

			for range ticker.C {
				// Randomly pick an item to buy and a random quantity
				item := ItemList[rand.Intn(len(ItemList))]
				quantity := rand.Intn(10) + 1 // Random quantity between 1 and 10
				traderID, err := String2Int(buyer.Trader)

				// Log the buyer's activity
				fmt.Printf("  Node %d (Buyer from Post %s): Sending buy request for %d units of %s to Trader %d", buyer.ID, buyer.Trader, quantity, item, traderID)

				if err != nil {
					log.Printf("Node %d: Error converting Trader ID: %v", buyer.ID, err)
					return
				}
				// Make the buy request
				buyer.BuyFromTrader(traderID, item, quantity)
			}
		}()

		reply.Type = "SUCCESS"
		reply.Message = fmt.Sprintf("Node %d (Buyer) timer started", buyer.ID)
		return nil
	}
	reply.Type = "FAILURE"
	reply.Message = "Node is not or no longer a buyer"
	return nil
}

/*
 * BuyFromTrader sends a buy request from a buyer to a trader.
 */
func (buyer *Node) BuyFromTrader(traderID int, item string, quantity int) {
	var request Message
	var reply Message

	// Prepare buy request
	request.From = buyer.ID
	request.To = traderID
	request.Type = "BUY"
	request.Item = item
	request.Quantity = quantity

	// Send buy request to the trader
	client, err := GetClient(traderID)
	if err != nil {
		log.Printf("Buyer %d: ERROR - Unable to connect to Trader %d: %v\n", buyer.ID, traderID, err)
		return
	}

	err = client.Call(fmt.Sprintf("Node%d.HandleBuyRequest", traderID), &request, &reply)
	if err != nil {
		log.Printf("Buyer %d: ERROR -  Error sending buy request to Trader %d: %v\n", buyer.ID, traderID, err)
	} else {
		fmt.Printf("  Buyer %d: Buy request response: %s\n", buyer.ID, reply.Message)
	}
	client.Close()
}

/*
 * HandleBuyRequest processes buy requests received by a trader from a buyer.
 */
func (trader *Node) HandleDepositRequest(request *Message, reply *Message) error {
	fmt.Printf("  Trader %d: Handling deposit request from Seller %d\n", trader.ID, request.From)

	// Forward the request to the warehouse (database server)
	client, err := GetClient(0) // Assuming warehouse is Node 0
	if err != nil {
		log.Printf("Trader %d: ERROR - Unable to connect to Warehouse: %v\n", trader.ID, err)
		reply.Type = "FAILURE"
		reply.Message = "Unable to connect to Warehouse"
		return err
	}
	defer client.Close()

	err = client.Call("Node0.SellProduct", request, reply)
	if err != nil {
		log.Printf("Trader %d: ERROR - Error forwarding deposit request to Warehouse: %v\n", trader.ID, err)
		reply.Type = "FAILURE"
		reply.Message = "Error forwarding deposit request"
		return err
	}

	reply.Type = "SUCCESS"
	log.Printf("Trader %d from Post%d: SUCCESS - Deposit request processed successfully\n", trader.ID, trader.Post)
	return nil
}

/*
 * HandleBuyRequest processes buy requests received by a trader from a buyer.
 */
func (trader *Node) HandleBuyRequest(request *Message, reply *Message) error {
	log.Printf("Trader %d: Handling buy request from Buyer %d\n", trader.ID, request.From)

	// Forward the request to the warehouse (database server)
	client, err := GetClient(0) // Getting the warehosue client
	if err != nil {
		log.Printf("Trader %d from Post%d: ERROR - Unable to connect to Warehouse: %v\n", trader.ID, trader.Post, err)
		reply.Type = "FAILURE"
		reply.Message = "Unable to connect to Warehouse"
		return err
	}
	defer client.Close()

	err = client.Call("Node0.BuyProduct", request, reply)
	if err != nil {
		log.Printf("Trader %d from Post%d: ERROR - Error forwarding buy request to Warehouse: %v\n", trader.ID, trader.Post, err)
		reply.Type = "FAILURE"
		reply.Message = "Error forwarding buy request"
		return err
	}

	// Respond to the buyer
	if reply.Type == "SUCCESS" {
		log.Printf("Trader %d from Post %d: Buy request successful: %s\n", trader.ID, trader.Post, reply.Message)
	} else {
		log.Printf("Trader %d from Post %d: Buy request failed: %s\n", trader.ID, trader.Post, reply.Message)
	}
	return nil
}

/*
 * BuyProduct processes the buy request received by the warehouse. This rpc method is triggered by the trader.
 */
func (w *Node) BuyProduct(req *Message, reply *Message) error {
	warehouse, ok := w.Role.(*Warehouse)
	if !ok {
		reply.Type = "FAILURE"
		reply.Message = "Node is not a warehouse"
		fmt.Printf("INFO - Node %d: Role is not a warehouse\n", w.ID)
		return fmt.Errorf("node %d: role is not a warehouse", w.ID)
	}

	warehouse.GlobalLock.Lock()
	defer warehouse.GlobalLock.Unlock()

	// Check if the item exists and has enough quantity
	if warehouse.Items[req.Item] >= req.Quantity {
		warehouse.Items[req.Item] -= req.Quantity
		reply.Type = "SUCCESS"
		reply.Message = fmt.Sprintf("Shipped %d units of %s", req.Quantity, req.Item)
		log.Printf("SUCEESS IN SHIPPING - Warehouse shipped %d units of %s. Remaining inventory: %v", req.Quantity, req.Item, warehouse.Items)
	} else {
		reply.Type = "FAILURE"
		reply.Message = fmt.Sprintf("Not enough %s in inventory. Available: %d", req.Item, warehouse.Items[req.Item])
		log.Printf("FAILED DUE TO INSUFFICIENT ITEMS - insufficient %s in Warehouse.  Available: %d", req.Item, warehouse.Items[req.Item])
	}

	// Save the updated inventory
	err := w.SaveInventoryToFile("inventory.txt")
	if err != nil {
		reply.Type = "FAILURE"
		reply.Message = "Failed to save inventory to file"
		log.Printf("ERROR - Warehouse saving inventory: %v", err)
		return err
	}

	return nil
}

/*
SellProduct processes the sell request received by the warehouse. This rpc method is triggered by the trader.
*/
func (w *Node) SellProduct(req *Message, reply *Message) error {
	warehouse, ok := w.Role.(*Warehouse)
	if !ok {
		reply.Type = "FAILURE"
		reply.Message = "Node is not a warehouse"
		log.Printf("Node %d: Role is not a warehouse", w.ID)
		return fmt.Errorf("node %d: role is not a warehouse", w.ID)
	}

	warehouse.GlobalLock.Lock()
	defer warehouse.GlobalLock.Unlock()

	// Add the item to the inventory
	warehouse.Items[req.Item] += req.Quantity
	reply.Type = "SUCCESS"
	reply.Message = fmt.Sprintf("Added %d units of %s to inventory", req.Quantity, req.Item)
	log.Printf("SUCCESS IN LOADING - Warehouse Added %d units of %s. Current inventory: %v", req.Quantity, req.Item, warehouse.Items)

	// Save the updated inventory
	err := w.SaveInventoryToFile("inventory.txt")
	if err != nil {
		reply.Type = "FAILURE"
		reply.Message = "Failed to save inventory to file"
		log.Printf("ERROR - Warehouse can not save in the inventory: %v", err)
		return err
	}

	return nil
}

/*
StartBuying starts the buyer node to send buy requests to the trader in every 10 seconds
*/
func (buyer *Node) StartBuying(traderID int) {
	go func() {
		ticker := time.NewTicker(10 * time.Second) // Adjust the frequency as needed
		defer ticker.Stop()

		for range ticker.C {
			item := ItemList[rand.Intn(len(ItemList))] // Randomly pick an item
			quantity := rand.Intn(10) + 1              // Random quantity (1-10)
			fmt.Printf("  Buyer %d from Post %d: Sending buy request for %d units of %s\n", buyer.ID, buyer.Post, quantity, item)
			buyer.BuyFromTrader(traderID, item, quantity)
		}
	}()

}
