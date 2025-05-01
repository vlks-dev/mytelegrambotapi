package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/mytelegrambot/service"
)

type BotHandler struct {
	service *service.Service
}

func NewBotHandler(service *service.Service) *BotHandler {
	return &BotHandler{service: service}
}

func (h *BotHandler) Commands(c *gin.Context) {
	commands, err := h.service.ListCommands(c)
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
