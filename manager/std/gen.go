package std

// these are the concurrent maps used in the manager
// requires carto: github.com/schigh/carto
//go:generate ../../bin/carto -lz -p std -s healthMap -k string -v *github.com/schigh/health/pkg/v1.Check -r m -o ./health_map.go
//go:generate ../../bin/carto -lz -p std -s checkerMap -k string -v wrapper -r m -o ./checker_map.go
//go:generate ../../bin/carto -lz -p std -s reporterMap -k string -v github.com/schigh/health.Reporter -r m -o ./reporter_map.go
//go:generate ../../bin/carto -lz -p std -s checkResultMap -k string -v result -r m -o ./check_result_map.go
//go:generate ../../bin/mockgen -package std_test -self_package github.com/schigh/health/managers/std_test -destination ./mocks_test.go github.com/schigh/health Checker,Reporter
