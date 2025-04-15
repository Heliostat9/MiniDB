package engine

func Execute(query string) (string, error) {
	res, err := HandleCommand(query)

	return res, err
}
