package market

func limitSlice[T any](items []T, limit int) []T {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func limitTail[T any](items []T, limit int) []T {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}
