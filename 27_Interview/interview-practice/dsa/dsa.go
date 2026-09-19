// Package dsa implements the LeetCode patterns most worth having fluent
// in Go, with the Go-specific detail each one hinges on.
package dsa

import "sort"

// TwoSum — the hash-map pattern. O(n) time, O(n) space.
func TwoSum(nums []int, target int) []int {
	seen := make(map[int]int, len(nums)) // pre-sized (Module 23)
	for i, n := range nums {
		if j, ok := seen[target-n]; ok {
			return []int{j, i}
		}
		seen[n] = i
	}
	return nil
}

// LongestUniqueSubstring — sliding window with a map of last-seen indices.
func LongestUniqueSubstring(s string) int {
	lastSeen := make(map[byte]int)
	best, start := 0, 0
	for i := 0; i < len(s); i++ {
		if j, ok := lastSeen[s[i]]; ok && j >= start {
			start = j + 1
		}
		lastSeen[s[i]] = i
		if i-start+1 > best {
			best = i - start + 1
		}
	}
	return best
}

// BinarySearch — note mid computed as lo + (hi-lo)/2, the overflow-safe
// habit interviewers look for even though Go's int is 64-bit.
func BinarySearch(nums []int, target int) int {
	lo, hi := 0, len(nums)-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		switch {
		case nums[mid] == target:
			return mid
		case nums[mid] < target:
			lo = mid + 1
		default:
			hi = mid - 1
		}
	}
	return -1
}

// Permutations — the classic BACKTRACKING slice-aliasing trap, handled
// correctly. See the guide: appending `current` directly would make every
// stored result alias the same backing array.
func Permutations(nums []int) [][]int {
	var results [][]int
	var current []int
	used := make([]bool, len(nums))

	var backtrack func()
	backtrack = func() {
		if len(current) == len(nums) {
			cp := make([]int, len(current)) // THE critical copy
			copy(cp, current)
			results = append(results, cp)
			return
		}
		for i, n := range nums {
			if used[i] {
				continue
			}
			used[i] = true
			current = append(current, n)
			backtrack()
			current = current[:len(current)-1] // undo
			used[i] = false
		}
	}
	backtrack()
	return results
}

// GroupAnagrams — map with a sorted-string key; shows Go's string/[]byte
// conversion, which interviewers often probe ("does this allocate?" — yes).
func GroupAnagrams(words []string) [][]string {
	groups := make(map[string][]string)
	for _, w := range words {
		b := []byte(w)
		sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })
		key := string(b)
		groups[key] = append(groups[key], w)
	}
	out := make([][]string, 0, len(groups))
	for _, g := range groups {
		out = append(out, g)
	}
	return out
}

// ReverseLinkedList — the pointer-manipulation classic.
type ListNode struct {
	Val  int
	Next *ListNode
}

func ReverseList(head *ListNode) *ListNode {
	var prev *ListNode
	for cur := head; cur != nil; {
		next := cur.Next
		cur.Next = prev
		prev = cur
		cur = next
	}
	return prev
}

// MaxSubArray — Kadane's algorithm, the canonical DP warm-up.
func MaxSubArray(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	best, current := nums[0], nums[0]
	for _, n := range nums[1:] {
		if current+n > n {
			current = current + n
		} else {
			current = n
		}
		if current > best {
			best = current
		}
	}
	return best
}
