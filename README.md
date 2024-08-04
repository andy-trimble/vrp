# Vehicle Route Problem for Pickup and Delivery (VRPPD)
Submitted by Andy Trimble

This is a solution to the VRPPD problem using the Clark-Wright Savings algorithm. In the spirit of full transparency, the implementation was adapted from [this](https://github.com/heet9022/Vehicle-Routing-Problem/tree/main) GitHub repository.

# Building and Running
This is a simple Go program that can be compiled by running `go build .`. This will produce a single binary named `vrp` in the current directory. It takes a single command line argument defining a file to read. For example, `./vrp <path>/<filename>`. Alternately, it may be executed by running `go run . <path>/<filename>`. However, running in this manner results in much slower execution times.

# References

[The Vehicle Routing Problem](https://www.researchgate.net/profile/Jacques-Desrosiers/publication/200622146_VRP_with_Pickup_and_Delivery/links/0deec528e7769dcf1d000000/VRP-with-Pickup-and-Delivery.pdf)

[Clark-Wright Savings Algorithm](https://web.mit.edu/urban_or_book/www/book/chapter6/6.4.12.html)
