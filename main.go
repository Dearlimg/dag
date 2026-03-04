package main

type Node struct {
	Name string
	g    *Graph
	ID   int64
	Mask int64
}

func NewNode() *Node {
	return &Node{}
}

func (n *Node) Enable() bool {
	return n.g != nil
}

func (n *Node) Run() {}

func (n *Node) Name() string {
	return "anode"
}
