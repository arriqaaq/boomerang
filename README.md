boomerang
================

This project is inspired by

	1) Go's default http package
	2) Jitter/Backoff ----> https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
	3) Hashicorp library https://github.com/hashicorp/go-retryablehttp

TODO
================

	1) Test cases for hystrix, basic testing for hystrix client
	2) More test cases for http client
	3) Add Prometheus metric support on the client


EXAMPLE USAGE
================

	1) HTTP Client
		client := NewHttpClient(10*time.Millisecond, DefaultTransport())
		backoffStratergy := NewExponentialBackoff(2*time.Millisecond, 10*time.Millisecond, 2.0)
		client.SetRetries(3)
		client.SetBackoff(backoffStratergy)


		resp, err := client.Get("/foo/bar")
		if err!=nil{
			resp.Body.Close()
		}

	2) Hystrix Client (TODO)
