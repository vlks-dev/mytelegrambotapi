package handlers

/*type BotHandler struct {
	service service.BotService
}

func NewBotHandler(service *service.Service) *BotHandler {
	return &BotHandler{service: service}
}

func (h *BotHandler) Commands(c *gin.Context) {
	commands, err := h.service.ListCommands(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
	}
	c.JSON(200, gin.H{"commands": commands})
}

func (h *BotHandler) RegisterRoutes(router *gin.Engine) {
	botGroup := router.Group("/bot")
	{
		botGroup.GET("/commands", h.Commands)
	}
}
*/
