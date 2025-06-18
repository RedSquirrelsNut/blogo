package cli

func RegisterAllCommands(c *Commands) {
	c.Register("login", HandlerLogin)
	c.Register("register", HandlerRegister)
	c.Register("reset", HandlerReset)
	c.Register("users", HandlerUsers)
	c.Register("agg", HandlerAgg)
	c.Register("browse", MiddlewareLoggedIn(HandlerBrowse))
	c.Register("addfeed", MiddlewareLoggedIn(HandlerAddFeed))
	c.Register("feeds", HandlerFeeds)
	c.Register("follow", MiddlewareLoggedIn(HandlerFollow))
	c.Register("following", MiddlewareLoggedIn(HandlerFollowing))
	c.Register("unfollow", MiddlewareLoggedIn(HandlerUnfollow))
}
