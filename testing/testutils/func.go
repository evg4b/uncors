package testutils

func Times(n int, function func()) {
	for i := 0; i < n; i++ {
		function()
	}
}
