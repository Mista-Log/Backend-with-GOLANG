package dsa

import (
	"reflect"
	"testing"
)

func TestTwoSum(t *testing.T) {
	tests := []struct {
		name   string
		nums   []int
		target int
		want   []int
	}{
		{"basic", []int{2, 7, 11, 15}, 9, []int{0, 1}},
		{"later pair", []int{3, 2, 4}, 6, []int{1, 2}},
		{"no solution", []int{1, 2}, 100, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TwoSum(tt.nums, tt.target); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TwoSum(%v, %d) = %v; want %v", tt.nums, tt.target, got, tt.want)
			}
		})
	}
}

func TestLongestUniqueSubstring(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"abcabcbb", 3},
		{"bbbbb", 1},
		{"pwwkew", 3},
		{"", 0},
	}
	for _, tt := range tests {
		if got := LongestUniqueSubstring(tt.input); got != tt.want {
			t.Errorf("LongestUniqueSubstring(%q) = %d; want %d", tt.input, got, tt.want)
		}
	}
}

func TestBinarySearch(t *testing.T) {
	nums := []int{1, 3, 5, 7, 9, 11}
	for _, target := range nums {
		if got := BinarySearch(nums, target); nums[got] != target {
			t.Errorf("BinarySearch(%d) returned index %d, which holds %d", target, got, nums[got])
		}
	}
	if got := BinarySearch(nums, 4); got != -1 {
		t.Errorf("BinarySearch for a missing value = %d; want -1", got)
	}
}

// TestPermutationsNoAliasing is the important one — it specifically
// guards against the backtracking slice-aliasing bug the guide warns
// about. If Permutations appended `current` directly instead of a copy,
// every result would be identical and this test would catch it.
func TestPermutationsNoAliasing(t *testing.T) {
	got := Permutations([]int{1, 2, 3})
	if len(got) != 6 {
		t.Fatalf("expected 6 permutations, got %d", len(got))
	}
	seen := make(map[string]bool)
	for _, p := range got {
		key := ""
		for _, n := range p {
			key += string(rune('0' + n))
		}
		if seen[key] {
			t.Errorf("duplicate permutation %v — likely a slice-aliasing bug", p)
		}
		seen[key] = true
	}
}

func TestMaxSubArray(t *testing.T) {
	tests := []struct {
		nums []int
		want int
	}{
		{[]int{-2, 1, -3, 4, -1, 2, 1, -5, 4}, 6},
		{[]int{1}, 1},
		{[]int{-3, -2, -5}, -2},
	}
	for _, tt := range tests {
		if got := MaxSubArray(tt.nums); got != tt.want {
			t.Errorf("MaxSubArray(%v) = %d; want %d", tt.nums, got, tt.want)
		}
	}
}
