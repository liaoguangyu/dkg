package dkg

import (
	"testing"
	"fmt"
	"math/big"
)

func Test(t *testing.T){
	var sij *SijShare
	var private *PriPoly
	var public *PubPoly

	private = SetPriShare(1,10)
	public = SetPubPolybyPrivate(private)
	sij = computeSij(private, 2)

	pub_byte := SerializePubPoly(public)
	sij_byte := SerializeSijShare(sij)

	fmt.Printf("public : %x\n", pub_byte)
	fmt.Printf("sij : %x\n", sij_byte)
}

func TestComputeSij(T *testing.T){
	var private PriPoly
	private.index = 2
	private.coeffs = make([]*big.Int,0)

	//init PriPoly
	for i := 0; i < 4; i++ {
		private.coeffs = append(private.coeffs, big.NewInt(1))
	}
	sij := new(big.Int)
	bignum := new(big.Int).SetInt64(2)
	bignum1 := new(big.Int)
	for i := 0; i < 4; i++ {
		bignum2 := new(big.Int).SetInt64(int64(i))
		bignum1.Exp(bignum, bignum2, p)
		sij.Add(sij, bignum1)
	}
	num := sij.Int64()
	fmt.Println(num)
}

func TestVerify(T *testing.T){
	var private PriPoly

	private.index = 2
	private.coeffs = make([]*big.Int,0)

	//init PriPoly
	for i := 0; i < 4; i++ {
		private.coeffs = append(private.coeffs, big.NewInt(1))
	}
	public := SetPubPolybyPrivate(&private)

	sij := computeSij(&private, 3)
	fmt.Printf("%s\n",sij.Sij.String())
	complain, _ := Verify(sij, public, 3)
	if complain == nil {
		fmt.Println("verify pass")
	}
}

func TestLagrangeBasis(T *testing.T){
	private := make([]*PriPoly, 5)
	public := make([]*PubPoly, 5)

	//init PriPoly
	for i := 0; i < 5; i++ {
		private[i] = new(PriPoly)
		private[i].index = i + 1
		private[i].coeffs = make([]*big.Int,0)
		for j := 0; j < 4; j++ {
			private[i].coeffs = append(private[i].coeffs, big.NewInt(1))
		}
	}

	for i := 0; i < 5; i++ {
		public[i] = SetPubPolybyPrivate(private[i])
	}

	bignum := LagrangeBase(3,public)
	fmt.Printf("bignum : %s\n",bignum.String())
}

func TestRecoverSignature(T *testing.T){
	
}

func TestRecoverKey(T *testing.T){
	
}
