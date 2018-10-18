package dkg

import (
	"golang.org/x/crypto/bn256"
	"fmt"
	"net"
	"os"
	"log"
	"encoding/gob"
	"bytes"
	"io/ioutil"
	"reflect"
	"time"
)

//node message
var shardingNode []string
//node connect message
var connectedNode = make(map[string]bool)
var connectsocket = make(map[string]net.Conn)

//channel for trigger operation
var indexchan = make(chan int, 1)

const COMPLAIN_TYPE = 0x00
const POLY_COEFFS_TYPE = 0x01
const SIJ_TYPE = 0X02
const RE_SIJ_TYPE = 0X03

type Complain struct {
	index int
	from int
	status bool
}
func SerializeComplain(complain *Complain)[]byte{

	return nil
}
func DeserializeComplain(complainbyte []byte) *Complain {
	var complain Complain

	decoder := gob.NewDecoder(bytes.NewReader(complainbyte))
	err := decoder.Decode(&complain)
	if err != nil {
		log.Panic(err)
	}

	return &complain
}

type Deal struct {
	keyshare *SijShare
	complain map[int]*Complain
	pubshare *PubPoly
	status bool
}
type Dealer struct{
	private *PriPoly
	deal map[int]*Deal
	index int
	threshold int
	timeout time.Duration
	complain int
	QUAL []int
}

func DistributedKeyGeneration(index, threshold, number int) (*bn256.G1,error){

	//start tcp node
	nodeID := os.Getenv("NODE_ID")
	node := fmt.Sprintf("localhost:%s", nodeID)

	tcpsocket, err := net.Listen("tcp", node)
	if err != nil {
		log.Panic(err)
	}
	defer tcpsocket.Close()

	//initialize private polynomial coefficient and public polynomial coefficient
	var dealer Dealer
	dealer.deal = make(map[int]*Deal)
	dealer.QUAL = make([]int, 100)

	dealer.private = SetPriShare(index, threshold)
	dealer.deal[index].pubshare = SetPubPolybyPrivate(dealer.private)
	dealer.deal[index].complain = make(map[int]*Complain)
	dealer.index = index
	dealer.threshold = threshold
	dealer.timeout = 5000
	dealer.complain = 0
	dealer.QUAL = append(dealer.QUAL, index)

	//connect to all node
	go connectToAllNode(indexchan)

	//broadcast public polynomial coefficient share
	go dealer.broadcast(dealer.deal[index].pubshare)

	//receive information from other participants
	go func(){
		for {
			conn, err := tcpsocket.Accept()
			if err != nil {
				log.Println("accept error: ", err)
			}
			message, err := ioutil.ReadAll(conn)
			if err != nil {
				log.Panic(err)
			}

			//split message type and message
			typename := message[0]
			msg := message[1:]

			//process msg
			switch typename {
			case COMPLAIN_TYPE:
				//process complain message
				complain := DeserializeComplain(msg)
				dealer.deal[complain.index].complain[index] = complain
			case POLY_COEFFS_TYPE:
				//process polynomial coefficient message
				pubshare := DeserializePubPoly(msg)
				if dealer.deal[pubshare.index].complain == nil {
					dealer.deal[pubshare.index].complain = make(map[int]*Complain)
				}

				if dealer.deal[pubshare.index].keyshare != nil {
					complain, _ := Verify(dealer.deal[pubshare.index].keyshare, pubshare, threshold)
					if complain != nil {
						dealer.broadcast(complain)
						continue
					}
					dealer.deal[pubshare.index].pubshare = pubshare
					dealer.QUAL = append(dealer.QUAL, pubshare.index)
					dealer.deal[pubshare.index].status = false
					continue
				}
			case SIJ_TYPE:
				//process sij message
				keyshare := DeserializeSijShare(msg)
				if dealer.deal[keyshare.index].complain == nil {
					dealer.deal[keyshare.index].complain = make(map[int]*Complain)
				}
				if dealer.deal[keyshare.index].pubshare != nil {
					complain, _ := Verify(keyshare, dealer.deal[keyshare.index].pubshare, dealer.index)
					if complain != nil {
						dealer.broadcast(complain)
						continue
					}
					dealer.deal[keyshare.index].keyshare = keyshare
					dealer.QUAL = append(dealer.QUAL, keyshare.index)
					dealer.deal[keyshare.index].status = false
					continue
				}
			case RE_SIJ_TYPE:
				keyshare := DeserializeSijShare(msg)
				complain, _ := Verify(keyshare, dealer.deal[keyshare.index].pubshare, dealer.index)
				if complain == nil {
					dealer.QUAL = append(dealer.QUAL, keyshare.index)
					dealer.complain -= 1
					if dealer.complain == 0 {
						dealer.compute()
					}
				}
			}
		}
	}()
	var timeout = make(chan bool, 1)
	go func(){
		time.Sleep(dealer.timeout)
		timeout <-true
	}()
	go func(){
		select{
			case <- timeout:
				dealer.computeQUAL(index)
			case sing_index := <- indexchan:
				//single cast to participant Pj
				go dealer.singlecast(sing_index)
		}
	}()
	return nil,nil
}

func connectToAllNode(index chan int){
	for i := 0; i < len(shardingNode); i++ {
		if !connectedNode[shardingNode[i]] {
			tcpsocket, err := net.Dial("tcp", shardingNode[i])
			if err != nil {
				log.Println("connect error: ",err)
				continue
			}
			connectsocket[shardingNode[i]] = tcpsocket
			index <- i
		}
	}
}

//broadcast message
func (dealer *Dealer)broadcast(share interface{}) error {
	var data, sharebyte []byte

	datatype := reflect.TypeOf(share)
	switch datatype.Name() {
	case "dkg.PubShare":
		data = append(data, POLY_COEFFS_TYPE)
		sharebyte = SerializePubPoly(nil)
	case "dkg.Complain":
		data = append(data, COMPLAIN_TYPE)
		sharebyte = SerializeComplain(nil)
	}
	data = append(data, sharebyte...)
	for i := 0; i < len(connectsocket); i++ {
		conn := connectsocket[shardingNode[i]]
		if conn != nil {
			conn.Write(data)
		}
	}
	return nil
}

//singlecast Sij to participant Pj
func (dealer *Dealer)singlecast(index int){
	var data []byte
	var Sij *SijShare
	data = append(data, SIJ_TYPE)
	conn := connectsocket[shardingNode[index]]
	if conn != nil {
		Sij = computeSij(dealer.private, index)
		data = append(data, SerializeSijShare(Sij)...)
		conn.Write(data)
	}
}

func (dealer *Dealer)computeQUAL(index int)*bn256.G1{
	var count int
	for i := 0; i < len(dealer.deal); i++ {
		count = len(dealer.deal[i].complain)
		if count == 0 {
			dealer.QUAL = append(dealer.QUAL, dealer.deal[i].pubshare.index)
		} else if count <= dealer.threshold {
			if  _, ok := dealer.deal[i].complain[index]; ok == false {
				dealer.complain += 1
				continue
			}
		} else if count > dealer.threshold {
			continue
		}

	}

	return nil
}

func (dealer *Dealer)compute()[]*bn256.G1{
	coeffs := make([]*bn256.G1, dealer.threshold)
	for i := 0; i < len(dealer.QUAL); i++ {
		for j := 0; j < dealer.threshold; j++ {
			coeffs[j] = coeffs[j].Add(coeffs[j], dealer.deal[i].pubshare.coeffs[j])
		}
	}

	return coeffs
}
