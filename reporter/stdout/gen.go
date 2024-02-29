package stdout

// these are the concurrent maps used in the manager
// requires carto: github.com/schigh/carto
//go:generate ../../bin/carto -lz -p stdout -s healthCheckMap -k string -v *github.com/schigh/health/pkg/v1.Check -r m -o ./check_map.go
