package stats

import (
	"math"
)

// PairedTTest performs a paired t-test on two sets of values.
// Returns the t-statistic and p-value.
func PairedTTest(before, after []float64) (tStat, pValue float64) {
	if len(before) != len(after) || len(before) == 0 {
		return 0, 1
	}

	n := float64(len(before))

	// Calculate differences
	diffs := make([]float64, len(before))
	for i := range before {
		diffs[i] = after[i] - before[i]
	}

	// Calculate mean difference
	meanDiff := mean(diffs)

	// Calculate standard error
	sumSqDev := 0.0
	for _, d := range diffs {
		sumSqDev += (d - meanDiff) * (d - meanDiff)
	}
	stdErr := math.Sqrt(sumSqDev / (n*(n-1)))

	if stdErr == 0 {
		return 0, 1 // No difference
	}

	tStat = meanDiff / stdErr

	// Calculate p-value using t-distribution with n-1 degrees of freedom
	df := n - 1
	pValue = tDistPValue(tStat, df)

	return tStat, pValue
}

// tDistPValue returns the two-tailed p-value for a t-statistic with df degrees of freedom.
// Uses the regularized incomplete beta function relation:
//   p = betainc(df/(df+t²), df/2, 0.5)
//
// For large df (> 30), uses normal approximation for efficiency.
func tDistPValue(t, df float64) float64 {
	absT := math.Abs(t)
	if absT == 0 {
		return 1
	}

	// Use normal approximation for large sample sizes
	if df > 30 {
		return 2 * (1 - normalCDF(absT))
	}

	// Use regularized incomplete beta function for small samples
	// x = df/(df + t²), a = df/2, b = 0.5
	x := df / (df + t*t)
	a := df / 2

	return betaincRegularized(x, a, 0.5)
}

// normalCDF returns the cumulative distribution function of standard normal.
func normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

// betaincRegularized computes the regularized incomplete beta function I_x(a, b)
// using a series expansion. This is used to compute t-distribution p-values.
func betaincRegularized(x, a, b float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}

	// Use symmetry transformation for better convergence when x > a/(a+b)
	// This uses the relation I_x(a,b) = 1 - I_{1-x}(b,a)
	if x > a/(a+b) && b < a {
		return 1 - betaincRegularized(1-x, b, a)
	}

	// Use the series expansion formula:
	// I_x(a,b) = (x^a / (a * B(a,b))) * sum_{n=0}∞ [(1-b)_n / (a+n)] * x^n
	// where (1-b)_n is the rising Pochhammer symbol (1-b)(2-b)...(n-b), (0)_0 = 1
	//
	// For b = 0.5, (1-b)_n = (0.5)_n = 0.5 * 1.5 * 2.5 * ... * (n-0.5)
	//
	// We use the recurrence:
	// term_n = term_{n-1} * x * (1-b+n-1) / (a+n) * a / (a+n-1) / n * (a+b+n-1) / (a+b+n-2)
	// But simpler: T_n/T_{n-1} = x * (n-b) / (n * (a+n-1)) * (a+b-1) / (a+b+n-1) * (a)/(a)
	// = x * (n-b) / n * 1/(a+n-1) * (a+b-1)/(a+b+n-1)

	// Compute the initial term using the beta function
	// B(a,b) = Gamma(a)*Gamma(b)/Gamma(a+b)
	// We use the relation: B(a,b) = (a+b)/(a) * B(a+1,b)
	// And: B(a+1,b) = a! * Gamma(b) / Gamma(a+b+1) which we can compute iteratively

	// For numerical stability, compute ln(B) using logGamma
	lnB := lnGamma(a) + lnGamma(b) - lnGamma(a+b)
	lnTerm := a*math.Log(x) - math.Log(a) - lnB
	term := math.Exp(lnTerm)
	result := term

	// Series expansion
	for n := 1; n < 200; n++ {
		// Update term using recurrence relation
		// T_n = T_{n-1} * x * (n-b) / (n * (a+n-1))
		term = term * x * (float64(n) - b) / (float64(n) * (a + float64(n) - 1))
		result += term

		// Check convergence
		if math.Abs(term) < 1e-12*math.Abs(result) || math.Abs(term) < 1e-15 {
			break
		}
	}

	return result
}

// lnGamma returns the natural logarithm of the gamma function.
// Uses the Stirling asymptotic series for z >= 0.5.
func lnGamma(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Use reflection for x < 0.5
	if x < 0.5 {
		return math.Log(math.Pi) - math.Log(math.Sin(math.Pi*x)) - lnGamma(1-x)
	}

	// Stirling series: lnGamma(z) ~ (z-0.5)*ln(z) - z + 0.5*ln(2π) + 1/(12z) - 1/(360z³) + 1/(1260z⁵) - ...
	z := x
	sum := (z-0.5)*math.Log(z) - z + 0.5*math.Log(2*math.Pi)
	sum += 1.0 / (12 * z)
	sum -= 1.0 / (360 * z * z * z)
	sum += 1.0 / (1260 * z * z * z * z * z)
	sum -= 1.0 / (1680 * z * z * z * z * z * z * z)
	sum += 1.0 / (1188 * z * z * z * z * z * z * z * z * z)
	return sum
}
