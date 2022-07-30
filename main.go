package main

func main() {
	var params Params

	params.init()

	req := newRequests(params)
	req.getDataByFiles()

}
