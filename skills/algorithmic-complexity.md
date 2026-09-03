---
name: algorithmic-complexity
description: Complexity bounds, algorithm preconditions that break silently, and the degenerate inputs to test first. From Algorithms Notes (GoalKicker).
---

# Algorithmic Complexity

## State the Bound You Mean

- **Worst case is the guarantee.** Average case needs an input distribution you can
  rarely prove; best case guarantees nothing.
- **Big-O is an upper bound; Big-Theta is tight.**
- Repeatedly halving the problem is O(log n).

## Structure Choice Dominates

Complexity follows the data structure, not the loop body: a map lookup is O(1) where a
scan is O(n); sorted data unlocks O(log n) search. Counting sort beats the comparison
bound at O(n+k) for small integer key ranges, but space grows with k.

Sorting has two orthogonal properties: **stability** and **in-place**. Merge sort is
Θ(n log n) always but needs O(n) auxiliary space. Quicksort degrades to O(n²) on sorted
input with a corner pivot; Hoare partitioning does far fewer swaps and handles
duplicate keys well.

## Preconditions That Break Silently

| Algorithm | Precondition |
|---|---|
| Binary search | Sorted / genuinely monotonic predicate |
| Dijkstra | Non-negative edge weights |
| Bellman-Ford | Use when negatives are possible; detects negative cycles, O(V·E) |
| DFS / BFS | Must mark visited or it loops forever |
| Greedy | Not globally optimal — coins {1,3,4} for 6 gives 4+1+1; optimal is 3+3 |
| Dynamic programming | Needs optimal substructure *and* overlapping subproblems |

Interval scheduling by earliest finish time is correct; earliest start, shortest
interval, and fewest conflicts all have counterexamples.

## Recurring Defects

- `(low + high) / 2` overflows — use `low + (high - low) / 2`.
- BST deletion's two-child case (in-order successor) is the classic bug; verify
  in-order traversal still yields sorted output.
- Deep recursion overflows the stack; memoizing fixes time, not depth.

## Prove Termination

Name a measure that strictly decreases and is bounded below. If you can't, you haven't
shown the loop terminates.

## Degenerate Inputs First

Empty, single element, all-equal, maximum size, pattern longer than input.
