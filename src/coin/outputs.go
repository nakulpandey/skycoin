package coin

import (
    "github.com/skycoin/skycoin/src/lib/encoder"
)

// Manages Unspents
type UnspentPool struct {
    // Points to a UxOut in Arr
    Map map[SHA256]int
    Arr []UxOut
    // Total running hash
    XorHash SHA256
}

func NewUnspentPool() *UnspentPool {
    return &UnspentPool{
        Arr:     make([]UxOut, 0),
        Map:     make(map[SHA256]int),
        XorHash: SHA256{},
    }
}

// Adds a UxOut to the UnspentPool
func (self *UnspentPool) Add(ux UxOut) {
    index := len(self.Arr)
    h := ux.Hash()
    self.Arr = append(self.Arr, ux)
    self.Map[h] = index
    self.XorHash.Xor(h)
}

// Returns a UxOut by hash, and whether it actually exists (if it does not
// exist, the map would return an empty UxOut)
func (self *UnspentPool) Get(h SHA256) (UxOut, bool) {
    i, ok := self.Map[h]
    return self.Arr[i], ok
}

// Returns true if an unspent exists for this hash
func (self *UnspentPool) Has(h SHA256) bool {
    _, ok := self.Map[h]
    return ok
}

// Removes an unspent from the pool, by hash
func (self *UnspentPool) Del(h SHA256) {
    i, ok := self.Map[h]
    if !ok {
        return
    }
    delete(self.Map, h)
    self.Arr = append(self.Arr[:i], self.Arr[i+1:]...)
    for j := i; j < len(self.Arr); j++ {
        // TODO -- store the UxOut hash in its header
        self.Map[self.Arr[j].Hash()] = j
    }
    self.XorHash.Xor(h)
}

// Delete multiple hashes in a batch
func (self *UnspentPool) DelMultiple(hashes []SHA256) {
    lowest := len(self.Arr)
    for _, h := range hashes {
        i, ok := self.Map[h]
        if !ok {
            continue
        }
        if i < lowest {
            lowest = i
        }
        delete(self.Map, h)
        self.Arr = append(self.Arr[:i], self.Arr[i+1:]...)
        self.XorHash.Xor(h)
    }
    for j := lowest; j < len(self.Arr); j++ {
        self.Map[self.Arr[j].Hash()] = j
    }
}

// Returns all Unspents for a single address
func (self *UnspentPool) AllForAddress(a Address) []UxOut {
    uxo := make([]UxOut, 0)
    for _, ux := range self.Arr {
        if ux.Body.Address == a {
            uxo = append(uxo, ux)
        }
    }
    return uxo
}

// Returns Unspents for multiple addresses
func (self *UnspentPool) AllForAddresses(addrs []Address) map[Address][]UxOut {
    m := make(map[Address]byte, len(addrs))
    for _, a := range addrs {
        m[a] = byte(1)
    }
    uxo := make(map[Address][]UxOut)
    for a, _ := range m {
        uxo[a] = make([]UxOut, 0)
    }
    for _, ux := range self.Arr {
        _, exists := m[ux.Body.Address]
        if exists {
            uxo[ux.Body.Address] = append(uxo[ux.Body.Address], ux)
        }
    }
    return uxo
}

/*
	Unspent Outputs
*/

//needs a nonce
//think through replay atacks

/*

- hash must only depend on factors known to sender
-- hash cannot depend on block executed
-- hash cannot depend on sequence number
-- hash may depend on nonce

- hash must depend only on factors known to sender
-- needed to minimize divergence during block chain forks
- it should be difficult to create outputs with duplicate ids

- Uxhash cannot depend on time or block it was created
- time is still needed for
*/

/*
	For each transaction, keep track of
	- order created
	- order spent (for rollbacks)
*/

type UxOut struct {
    Head UxHead
    Body UxBody //hashed part
    //Meta UxMeta
}

//not hashed, metdata
type UxHead struct {
    Time  uint64 //time of block it was created in
    BkSeq uint64 //block it was created in
    SpSeq uint64 //block it was spent in
}

//part that is hashed
type UxBody struct {
    SrcTransaction SHA256
    Address        Address //address of receiver
    Coins          uint64  //number of coins
    Hours          uint64  //coin hours
}

//type UxMeta struct {
//}

func (self *UxOut) Hash() SHA256 {
    b1 := encoder.Serialize(self.Body)
    return SumSHA256(b1)
}

/*
func (self UxOut) HashTotal() *SHA256 {
	b1 := encoder.Serialize(self.Head)
	b2 := encoder.Serialize(self.Body)
	b3 = append(b1, b2...)
	return SumSHA256(b3)
}
*/

/*
	Make indepedent of block rate?
	Then need creation time of output
	Creation time of transaction cant be hashed
*/

//calculate coinhour balance of output. t is the current unix utc time
func (self *UxOut) CoinHours(t uint64) uint64 {
    if t < self.Head.Time {
        return 0
    }

    v1 := self.Body.Hours             //starting coinshour
    ch := (t - self.Head.Time) / 3600 //number of hours, one hour every 240 block
    v2 := ch * self.Body.Coins        //accumulated coin-hours
    return v1 + v2                    //starting+earned
}
