package blastradius

// inboundAdjacency maps to_file_id -> files that import it (dependents).
func inboundAdjacency(deps []struct{ from, to int64 }) map[int64][]int64 {
	inbound := make(map[int64][]int64)
	for _, d := range deps {
		inbound[d.to] = append(inbound[d.to], d.from)
	}
	return inbound
}

// bfsDependents returns file IDs that depend on targetID within maxDepth hops.
// Each file is recorded at its shallowest depth; depth 1 = direct importers.
func bfsDependents(targetID int64, inbound map[int64][]int64, maxDepth int) map[int64]int {
	if maxDepth < 1 {
		return nil
	}
	depthOf := make(map[int64]int)
	queue := make([]int64, 0, 16)
	for _, dep := range inbound[targetID] {
		if _, seen := depthOf[dep]; seen {
			continue
		}
		depthOf[dep] = 1
		queue = append(queue, dep)
	}
	for qi := 0; qi < len(queue); qi++ {
		cur := queue[qi]
		d := depthOf[cur]
		if d >= maxDepth {
			continue
		}
		for _, next := range inbound[cur] {
			if _, seen := depthOf[next]; seen {
				continue
			}
			depthOf[next] = d + 1
			queue = append(queue, next)
		}
	}
	return depthOf
}
