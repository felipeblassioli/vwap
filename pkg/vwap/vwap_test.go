package vwap

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculator_Update(t *testing.T) {
	t.Run("Data point parse failure", func(t *testing.T) {
		tests := []struct {
			price    string
			quantity string
		}{
			{"not-a-float", "3.14159265359"},
			{"3.14159265359", "not-a-float"},
			{"1.13210a", "3.14159265359"},
			{"3.14159265359", "1.13210.a"},
			{"1,625", "3.14159265359"},
			{"3.14159265359", "1,625"},
		}

		const irrelevant = 1
		for _, tc := range tests {
			calc := NewCalculator(irrelevant)

			_, err := calc.Update(tc.price, tc.quantity)

			assert.ErrorIs(t, err, ErrFloatParse)
		}
	})

	t.Run("VWAP Calculation", func(t *testing.T) {
		tests := []struct {
			name        string
			windowWidth int
			prices      []string
			quantities  []string
			vwaps       []string
		}{
			{
				"Identity: window width 1",
				1,
				[]string{"1", "2", "3", "4", "5", "6"},
				[]string{"2", "3", "5", "7", "11", "13"},
				[]string{"1", "2", "3", "4", "5", "6"},
			},
			{
				"Ascending Pairs: window width 2",
				2,
				[]string{"1", "2", "3", "4", "5", "6"},
				[]string{"2", "3", "5", "7", "11", "13"},
				// VWAPs: 1, 8/5, 21/8, 43/12, 83/18,
				[]string{"1", "1.6", "2.625", "3.5833333333333335", "4.611111111111111", "5.541666666666667"},
			},
			{
				"No sliding: window width equals num. of data points",
				6,
				[]string{"1", "2", "3", "4", "5", "6"},
				[]string{"2", "3", "5", "7", "11", "13"},
				// VWAPs: 1, 8/5, 23/10, 51/17, 106/28, 184/41
				[]string{"1", "1.6", "2.3", "3", "3.7857142857142856", "4.487804878048781"},
			},
			{
				"No sliding: window width bigger than num. of data points",
				7,
				[]string{"1", "2", "3", "4", "5", "6"},
				[]string{"2", "3", "5", "7", "11", "13"},
				// VWAPs: 1, 8/5, 23/10, 51/17, 106/28, 184/41
				[]string{"1", "1.6", "2.3", "3", "3.7857142857142856", "4.487804878048781"},
			},
			{
				"Descending pairs: window width 2",
				2,
				[]string{"6", "5", "4", "3", "2", "1"},
				[]string{"2", "3", "5", "7", "11", "13"},
				// VWAPs: 1, 27/5, 35/8, 41/12, 43/18, 35/24
				[]string{"6", "5.4", "4.375", "3.4166666666666665", "2.388888888888889", "1.4583333333333333"},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				calc := NewCalculator(tc.windowWidth)

				for i := 0; i < len(tc.prices); i++ {
					vwap, _ := calc.Update(tc.prices[i], tc.quantities[i])

					exp, _ := new(big.Float).SetPrec(prec).SetMode(mode).SetString(tc.vwaps[i])
					act, _ := new(big.Float).SetPrec(prec).SetMode(mode).SetString(vwap)

					assert.True(t, exp.Cmp(act) == 0, "VWAPs %v !== %v", vwap, tc.vwaps[i])
				}
			})
		}
	})
}
