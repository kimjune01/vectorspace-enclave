package auction

import (
	"math"
	"testing"
)

// σ = 0 is the keyword limit: full score at the exact point, -Inf elsewhere.
// See: june.kim/keywords-are-tiny-circles

func TestSigmaZeroScoresAtExactPoint(t *testing.T) {
	point := []float64{0.1, 0.2, 0.3}
	got := ComputeEmbeddingScore(2.0, point, 0, point)
	want := math.Log(2.0) / math.Log(LogBase)
	if math.Abs(got-want) > 1e-12 {
		t.Errorf("exact point: got %v, want %v", got, want)
	}
}

func TestSigmaZeroLosesOffPoint(t *testing.T) {
	point := []float64{0.1, 0.2, 0.3}
	query := []float64{0.1, 0.2, 0.3000001}
	if got := ComputeEmbeddingScore(100.0, point, 0, query); !math.IsInf(got, -1) {
		t.Errorf("off point: got %v, want -Inf", got)
	}
}

func TestKeywordBidBeatsRicherVectorBidAtItsPoint(t *testing.T) {
	point := []float64{0.1, 0.2, 0.3}
	keyword := CoreBid{ID: "kw", Bidder: "kw-adv", Price: 1.0, Embedding: point, Sigma: 0}
	vector := CoreBid{ID: "vec", Bidder: "vec-adv", Price: 50.0, Embedding: []float64{0.9, 0.9, 0.9}, Sigma: 0.3}

	result := RunAuction([]CoreBid{keyword, vector}, 0, point)
	if result.Winner == nil || result.Winner.ID != "kw" {
		t.Fatalf("winner = %+v, want keyword bid", result.Winner)
	}
	if result.RunnerUp == nil || result.RunnerUp.ID != "vec" {
		t.Fatalf("runner-up = %+v, want vector bid", result.RunnerUp)
	}
	payment := ComputeVCGPayment(result, point, 0)
	if payment <= 0 || payment > vector.Price {
		t.Errorf("payment = %v, want in (0, %v]", payment, vector.Price)
	}
}

func TestSigmaZeroOffPointCannotWin(t *testing.T) {
	keyword := CoreBid{ID: "kw", Bidder: "kw-adv", Price: 100.0, Embedding: []float64{0.1, 0.2, 0.3}, Sigma: 0}
	result := RunAuction([]CoreBid{keyword}, 0, []float64{0.5, 0.5, 0.5})
	if result.Winner != nil {
		t.Errorf("winner = %+v, want nil (keyword away from its point)", result.Winner)
	}
}

func TestVCGPaymentBothExactMatch(t *testing.T) {
	point := []float64{0.1, 0.2, 0.3}
	a := CoreBid{ID: "a", Bidder: "a-adv", Price: 5.0, Embedding: point, Sigma: 0}
	b := CoreBid{ID: "b", Bidder: "b-adv", Price: 2.0, Embedding: point, Sigma: 0}

	result := RunAuction([]CoreBid{a, b}, 0, point)
	if result.Winner == nil || result.Winner.ID != "a" {
		t.Fatalf("winner = %+v, want a (higher price at same point)", result.Winner)
	}
	payment := ComputeVCGPayment(result, point, 0)
	if math.Abs(payment-2.0) > 1e-12 {
		t.Errorf("payment = %v, want 2.0 (pure second price when both exact)", payment)
	}
}

func TestSoloWinnerPaysReserve(t *testing.T) {
	point := []float64{0.1, 0.2, 0.3}
	solo := CoreBid{ID: "a", Bidder: "a-adv", Price: 5.0, Embedding: point, Sigma: 0.5}

	result := RunAuction([]CoreBid{solo}, 1.5, point)
	if result.Winner == nil {
		t.Fatal("want a winner")
	}
	if payment := ComputeVCGPayment(result, point, 1.5); payment != 1.5 {
		t.Errorf("payment = %v, want 1.5 (the reserve, not the bid)", payment)
	}
	if payment := ComputeVCGPayment(result, point, 0); payment != 0 {
		t.Errorf("payment = %v, want 0 (no reserve, no competitor)", payment)
	}
}
