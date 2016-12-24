package spn

// Node represents a node in an SPN.
type Node struct {
	// Parent nodes.
	pa []SPN
	// Children nodes.
	ch []SPN
	// Scope of this node.
	sc []int
	// Stores last soft inference values.
	s float64
	// Stores partial derivatives wrt parent.
	pnode float64
	// Stores the top part S(y,1,x) of the conditional value S(y,1|x) = S(y,1,x)/S(x).
	st float64
	// Stores the bottom part S(x) of the conditional value S(y,1|x) = S(y,1,x)/S(x).
	sb float64
	// Stores the conditional value S(y,1|x).
	scnd float64
	// Signals this node to be the root of the SPN.
	root bool
}

// An SPN is a node.
type SPN interface {
	// Value returns the value of this node given an instantiation.
	Value(val VarSet) float64
	// Max returns the MAP value of this node given an evidence.
	Max(val VarSet) float64
	// ArgMax returns the MAP value and state given an evidence.
	ArgMax(val VarSet) (VarSet, float64)
	// Ch returns the set of children of this node.
	Ch() []SPN
	// Pa returns the set of parents of this node.
	Pa() []SPN
	// Sc returns the scope of this node.
	Sc() []int
	// Type returns the type of this node.
	Type() string
	// AddChild adds a child to this node.
	AddChild(c SPN)
	// AddParent adds a parent to this node.
	AddParent(p SPN)
	// Stored returns the last stored soft inference value.
	Stored() float64
	// Derivative returns the partial derivative wrt its parent.
	Derivative() float64
	// Derive recursively derives this node and its children based on the last inference value.
	Derive()
	// Rootify signalizes this node is a root. The only change this does is set pnode=1.
	Rootify()
	// GenUpdate generatively updates weights given an eta learning rate.
	GenUpdate(eta float64)
	// DrvtAddr returns the address of the derivative for easier updating.
	DrvtAddr() *float64
	// Common base for all soft inference methods.
	Bsoft(val VarSet, where *float64) float64
	// Normalizes the SPN.
	Normalize()
	// CondValue returns the value of this SPN queried on Y and conditioned on X.
	CondValue(Y VarSet, X VarSet) float64
	// StoredTop returns the last top valuation S(X,Y) of S(Y|X).
	StoredTop() float64
	// StoredBottom returns the last bottom valuation S(X) of S(Y|X).
	StoredBottom() float64
	// DiscUpdate discriminatively updates weights given an eta learning rate.
	DiscUpdate(eta float64)
	// ResetDP resets all stored values. At the next inference call, the SPN will recompute
	// everything.
	ResetDP()
}

// VarSet is a variable set specifying variables and their respective instantiations.
type VarSet map[int]int

// Value returns the value of this node given an instantiation. (virtual)
func (n *Node) Value(val VarSet) float64 {
	return -1
}

// Max returns the MAP value of this node given an evidence. (virtual)
func (n *Node) Max(val VarSet) float64 {
	return -1
}

// ArgMax returns the MAP value and state given an evidence. (virtual)
func (n *Node) ArgMax(val VarSet) (VarSet, float64) {
	return nil, -1
}

// Ch returns the set of children of this node.
func (n *Node) Ch() []SPN {
	return n.ch
}

// Pa returns the set of parents of this node.
func (n *Node) Pa() []SPN {
	return n.pa
}

// Sc returns the scope of this node.
func (n *Node) Sc() []int {
	return n.sc
}

// Type returns the type of this node.
func (n *Node) Type() string {
	return "node"
}

// AddChild adds a child to this node.
func (n *Node) AddChild(c SPN) {
	n.ch = append(n.ch, c)
	c.AddParent(n)
}

// AddParent adds a parent to this node.
func (n *Node) AddParent(p SPN) {
	n.pa = append(n.pa, p)
}

// Stored returns the last stored soft inference value.
func (n *Node) Stored() float64 {
	return n.s
}

// Derivative returns the derivative of this node.
func (n *Node) Derivative() float64 {
	return n.pnode
}

// Derive recursively derives this node and its children based on the last inference value.
func (n *Node) Derive() {}

// Rootify signalizes this node is a root. The only change this does is set pnode=1.
func (n *Node) Rootify() {
	n.pnode = 1
	n.root = true
}

// GenUpdate generatively updates weights given an eta learning rate.
func (n *Node) GenUpdate(eta float64) {
	m := len(n.ch)

	for i := 0; i < m; i++ {
		n.ch[i].GenUpdate(eta)
	}
}

// DrvtAddr returns the address of the derivative for easier updating.
func (n *Node) DrvtAddr() *float64 { return &n.pnode }

// Bsoft is a common base for all soft inference methods.
func (n *Node) Bsoft(val VarSet, where *float64) float64 { return -1 }

// Normalize normalizes the SPN's weights.
func (n *Node) Normalize() {
	m := len(n.ch)

	for i := 0; i < m; i++ {
		n.ch[i].Normalize()
	}
}

// CondValue returns the value of this SPN queried on Y and conditioned on X.
// Let S be this SPN. If S is the root node, then CondValue(Y, X) = S(Y|X). Else we store the value
// of S(Y, X) in Y so that we don't need to recompute Union(Y, X) at every iteration.
func (n *Node) CondValue(Y VarSet, X VarSet) float64 { return -1 }

// StoredTop returns the last top valuation S(X,Y) of S(Y|X).
func (n *Node) StoredTop() float64 { return n.st }

// StoredBottom returns the last bottom valuation S(X) of S(Y|X).
func (n *Node) StoredBottom() float64 { return n.sb }

// ResetDP resets all dynamic programming stored values. At the next inference call, the SPN will
// recompute everything.
func (n *Node) ResetDP() {
	n.s = -1
	n.st = -1
	n.sb = -1
	n.scnd = -1
}

// DiscUpdate discriminatively updates weights given an eta learning rate.
func (n *Node) DiscUpdate(eta float64) {
	m := len(n.ch)

	for i := 0; i < m; i++ {
		n.ch[i].DiscUpdate(eta)
	}
}
