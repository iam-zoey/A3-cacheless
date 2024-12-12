package shared

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

/*
 * DepositItems processes deposit requests received by a trader from a seller.
 */
func (trader *Node) DepositItems(request *Message, reply *Message) error {
	trader.Mutex.Lock()
	defer trader.Mutex.Unlock()

	fmt.Printf("Trader (Node %d) received deposit request from Seller (Node %d): %d %s\n",
		trader.ID, request.From, request.Quantity, request.Item)

	// Process the request
	reply.Type = "SUCCESS" // Example: Accept all deposits
	fmt.Printf("  Trader (Node %d) accepted the deposit: %d %s\n", trader.ID, request.Quantity, request.Item)
	return nil
}

/*
StartSeller starts the seller node to generate items every 10 seconds.
*/
func (n *Node) StartSeller(args *Message, reply *Message) error {
	if seller, ok := n.Role.(*Seller); ok {
		// Start generating items every 10 seconds (Tg)
		// and generate 10 units (Ng) of items
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				item := ItemList[rand.Intn(len(ItemList))]
				seller.ItemsForSale[item] += 10
				fmt.Printf("INFO - Node %d (Seller from Post %d): Generated 10 units of %s. Total inventory: %v\n", n.ID, n.Post, item, seller.ItemsForSale)
				n.SendLoadRequestToTrader()

			}
		}()
		reply.Type = "SUCCESS"
		reply.Message = fmt.Sprintf("Node %d (Seller) timer started", n.ID)
		return nil
	}
	reply.Type = "FAILURE"
	reply.Message = "Node is not or no longer a seller"
	return nil
}

/**
 * SendLoadRequestToTrader sends a load request from a seller to a trader.
 * It randomly selects an item and quantity from the seller's inventory to send to the trader.
 */
func (n *Node) SendLoadRequestToTrader() {
	if seller, ok := n.Role.(*Seller); ok {
		traderID, err := String2Int(n.Trader)
		if err != nil {
			log.Printf("ERROR - Node %d (Seller): Error converting Trader ID: %v", n.ID, err)
			return
		}
		client, err := GetClient(traderID)
		if err != nil {
			log.Printf("ERROR - Node %d (Seller): Unable to connect to Trader %d: %v", n.ID, traderID, err)
			return
		}
		defer client.Close()

		var req Message
		var reply Message

		req.From = n.ID
		req.To = traderID
		req.Type = "LOAD"
		req.Item = ItemList[rand.Intn(len(ItemList))]
		req.Quantity = seller.ItemsForSale[req.Item]

		err = client.Call(fmt.Sprintf("Node%d.HandleLoadRequest", traderID), &req, &reply)
		if err != nil {
			log.Printf("ERROR - Node %d (Seller): Failed to send load request to Trader %d: %v", n.ID, traderID, err)
			return
		}

		if reply.Type == "SUCCESS" {
			log.Printf("SUCCESS - Node %d (Seller from Post %d): Loaded %d units of %s to Trader %d", n.ID, n.Post, req.Quantity, req.Item, traderID)
			delete(seller.ItemsForSale, req.Item)
		} else {
			log.Printf("REJECTED - Node %d (Seller from Post %d): Failed to load goods to Trader %d: %s", n.ID, n.Post, traderID, reply.Message)
		}
	}
}

/*
 * HandleLoadRequest processes load requests received by a trader from a seller.
 * It forwards the request to the warehouse for inventory update.
 */
func (n *Node) HandleLoadRequest(req *Message, reply *Message) error {
	if _, ok := n.Role.(*Trader); ok {
		fmt.Printf("  Node %d (Trader from Post %d): Received load request from Seller %d: %d units of %s", n.ID, n.Post, req.From, req.Quantity, req.Item)

		// Forward the request to the warehouse
		var warehouseReq Message
		var warehouseReply Message
		warehouseReq.Type = "LOAD"
		warehouseReq.Item = req.Item
		warehouseReq.Quantity = req.Quantity

		err := n.ForwardToWarehouse(&warehouseReq, &warehouseReply)
		if err != nil {
			reply.Type = "FAILURE"
			reply.Message = fmt.Sprintf("Failed to forward to warehouse: %v", err)
			return err
		}

		// Respond to the seller based on warehouse's response
		reply.Type = warehouseReply.Type
		reply.Message = warehouseReply.Message
		fmt.Printf("  Node %d (Trader from Post %d): Forwarded load request to warehouse. Response: %s", n.ID, n.Post, warehouseReply.Message)
		return nil
	}

	reply.Type = "FAILURE"
	reply.Message = "Node is not a trader"
	return nil
}

/**
 * ForwardToWarehouse forwards a request from a trader to the warehouse node.
 */
func (n *Node) ForwardToWarehouse(req *Message, reply *Message) error {
	client, err := GetClient(0) // Assuming the warehouse is Node 0
	if err != nil {
		log.Printf("ERROR-Node %d: Unable to connect to warehouse: %v", n.ID, err)
		return err
	}
	defer client.Close()

	time.Sleep(5 * time.Second)

	err = client.Call(fmt.Sprintf("Node%d.UpdateInventory", 0), req, reply)
	if err != nil {
		log.Printf("ERROR - Node %d: Error communicating with warehouse: %v", n.ID, err)
		return err
	}

	return nil
}
