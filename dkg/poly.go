package dkg

import (
	"golang.org/x/crypto/bn256"
	"math/big"
	"bytes"
	"encoding/gob"
	"log"
	"math/rand"
	"time"
	"fmt"
)

type PriPoly struct{
	index int
	coeffs []*big.Int
}

type PubPoly struct{
	index int
	coeffs []*bn256.G1
}

type SijShare struct{
	index int
	Sij *big.Int
}

var s = string("65000549695646603732796438742359905742825358107623003571877145026864184071783")
var QUAL []int
var p, _ = new(big.Int).SetString(s,10)

//marshal struct
func SerializeSijShare(sij *SijShare) []byte {
	var data [][]byte
	data = append(data, IntToByte(int64(sij.index)))
	buf, err := sij.Sij.MarshalJSON()
	if err != nil {
		log.Panic("*big.Int marshal error")
	}
	data = append(data, buf)
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err = encoder.Encode(sij)
	if err != nil {
		log.Panic(err)
	}
	return result.Bytes()
}

func DeserializeSijShare(sharebyte []byte) *SijShare {
	var data [][]byte
	var sij SijShare

	decoder := gob.NewDecoder(bytes.NewReader(sharebyte))
	err := decoder.Decode(&data)
	if err != nil {
		log.Panic(err)
	}

	sij.index = ByteToInt(data[0])
	sij.Sij = new(big.Int)
	sij.Sij.UnmarshalJSON(data[1])

	return &sij
}

func SerializePubPoly(pubpoly *PubPoly) []byte {

	var data [][]byte
	data = append(data, IntToByte(int64(pubpoly.index)))
	for i := 0; i < len(pubpoly.coeffs); i++{
		buf := pubpoly.coeffs[i].Marshal()
		data = append(data, buf)
	}

	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(&data)
	if err != nil {
		log.Panic(err)
	}
	return result.Bytes()
}

func DeserializePubPoly(sharebyte []byte) *PubPoly {
	var public PubPoly
	var data [][]byte

	decoder := gob.NewDecoder(bytes.NewReader(sharebyte))
	err := decoder.Decode(&data)
	if err != nil {
		log.Panic(err)
	}
	public.index = ByteToInt(data[0])
	public.coeffs = make([]*bn256.G1, 0)

	for i := 1; i < len(data); i++{
		point := new(bn256.G1)
		point.Unmarshal(data[i])
		public.coeffs = append(public.coeffs,point)
	}

	return &public
}

func SetPriShare(index, threshold int)*PriPoly{
	var private PriPoly
	private.coeffs = make([]*big.Int, 0)
	private.index = index
	for i := 0; i < threshold; i++{
		rand := rand.New(rand.NewSource(time.Now().Unix()))
		bignum := new(big.Int).Rand(rand, p)
		private.coeffs = append(private.coeffs, bignum)
		fmt.Printf("cap : %d\n", cap(private.coeffs))
		fmt.Printf("len : %d\n", len(private.coeffs))
	}

	return &private
}

func SetPubPolybyPrivate(private *PriPoly)*PubPoly{
	var public PubPoly
	public.coeffs = make([]*bn256.G1, 0)

	length := len(private.coeffs)

	for i := 0; i < length; i++{
		g1 := new(bn256.G1).ScalarBaseMult(private.coeffs[i])
		public.coeffs = append(public.coeffs,g1)
	}
	public.index = private.index

	return &public
}

func computeSij(pripoly *PriPoly, index int)*SijShare{
	var Sij *big.Int
	var bignum1, bignum2 *big.Int

	bignum1 = new(big.Int)
	bignum2 = new(big.Int)

	Sij = pripoly.coeffs[0]
	fmt.Printf("%s\n",Sij.String())
	bignum1.SetInt64(int64(index))

	for i := 1; i < len(pripoly.coeffs); i++ {
		bignum2.SetInt64(int64(i))
		bignum2 = bignum2.Exp(bignum1, bignum2, p)
		Sij = Sij.Add(Sij, bignum2)
		fmt.Printf("%s\n",Sij.String())
	}

	return &SijShare{pripoly.index,Sij }
}

func Verify(Sij *SijShare,pubShare *PubPoly,index int)(*Complain,error){
	var complain  Complain
	complain.index = index
	complain.from = pubShare.index
	complain.status = false

	bignum1 := new(big.Int).SetInt64(int64(index))
	bignum2 := new(big.Int)
	bignum3 := new(big.Int)

	g1 := pubShare.coeffs[0]
	g2 := new(bn256.G1).ScalarBaseMult(Sij.Sij)

	for i := 1; i < len(pubShare.coeffs); i++{
		bignum2.SetInt64(int64(i))
		bignum3.Exp(bignum1,bignum2,p)

		g := new(bn256.G1).ScalarMult(pubShare.coeffs[i], bignum3)
		g1 = g1.Add(g,g1)
	}

	byte1 := g1.Marshal()
	byte2 := g2.Marshal()
	fmt.Println(byte1)
	fmt.Println(byte2)
	result := bytes.Compare(byte1, byte2)

	fmt.Println("compare")
	if result == 0 {
		complain.status = true
		fmt.Println("compare pass")
		return nil, nil
	}

	return &complain, nil
}

//compute lj(x)
func LagrangeBase(index int,pubshare []*PubPoly)(*big.Int){
	bignum1 := new(big.Int).SetInt64(-1)
	bignum2 := new(big.Int).SetInt64(int64(index))
	bignum3 := new(big.Int).SetInt64(1)
	bignum4 := new(big.Int).SetInt64(1)
	bignum5 := new(big.Int).SetInt64(1)
	bignum6 := new(big.Int).SetInt64(1)

	for i := 0; i < len(pubshare); i++{
		if index == pubshare[i].index {
			continue
		}
		bignum3.SetInt64(int64(pubshare[i].index))		//xj
		bignum3.Mul(bignum1,bignum3)					//-xm
		bignum4.Add(bignum3,bignum2)					//xj-xm


		bignum5.Mul(bignum5,bignum3)					//-xm/(xj-xm)
		bignum6.Mul(bignum6,bignum4)
	}
	bignum5.Div(bignum5, bignum6)

	return bignum5
}

//compute group signature
/*func RecoverSignature(pubshare []PubPoly,signature [][]byte)[]byte{
	var point []*bn256.G1
	var sigpoint *bn256.G1
	var sigbyte []byte
	var bignum []*big.Int
	var check bool

	for i := 0; i < len(signature); i++ {
		point[i], check = point[i].Unmarshal(signature[i])
		if check ==false && point[i] == nil {
			i--
		}
	}

	for i := 0; i < len(pubshare); i++ {
		bignum[i] = LagrangeBase(pubshare[i].index, pubshare)
	}

	for i := 0; i < len(signature); i++ {
		point[i].ScalarMult(point[i], bignum[i])
		sigpoint = sigpoint.Add(sigpoint,point[i])
	}

	sigbyte = sigpoint.Marshal()

	return sigbyte
}
*/
//compute group public key and public polynomial coefficient
/*func RecoverKey(pubshare []PubPoly, threshold int)(*bn256.G1, []*bn256.G1){
	var groupkey *bn256.G1
	var coeffs []*bn256.G1
	var length int = len(pubshare[0].coeffs)

	//compute group public key
	for i := 0; i < len(pubshare); i++ {
		groupkey = groupkey.Add(groupkey,pubshare[i].coeffs[0])
	}

	//compute public polynomial coefficient
	for i := 1; i <len(pubshare); i++ {
		for j := 0; j < length; j++ {
			coeffs[i] = coeffs[i].Add(coeffs[i], pubshare[i].coeffs[j])
		}
	}

	return groupkey, coeffs
}*/




