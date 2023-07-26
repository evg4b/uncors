package testutils

func Times(n int, function func(n int)) {
	for i := 0; i < n; i++ {
		function(i)
	}
}
