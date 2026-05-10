// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive

// editDistance returns the Levenshtein distance between a and b — the
// minimum number of single-character insertions, deletions, or
// substitutions to transform one into the other.
//
// Used by [Registry.Suggest] to compute closest-match suggestions for
// "did you mean?" diagnostics. The implementation is the classic
// dynamic-programming variant with rolling rows (O(len(a)*len(b))
// time, O(len(b)) space). For the directive-name lengths we
// encounter (typically <30 chars), it's negligible.
func editDistance(a, b string) int {
	la, lb := len(a), len(b)
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := range lb + 1 {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// min3 returns the smallest of three ints.
func min3(a, b, c int) int { return min(a, min(b, c)) }
