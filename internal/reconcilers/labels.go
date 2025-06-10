package reconcilers

func labelsForDocling(name string) map[string]string {
	return map[string]string{"app": "docling-serve", "doclingserve_cr": name}
}
