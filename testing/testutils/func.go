package testutils

func Times(n int, function func(n int)) {
	for i := range n {
		function(i)
	}
}
