package voter

import (
	"fmt"
	"log"
	"time"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// Voter handles scheduled checking and processing of expired resource votes
type Voter struct {
	session  *discordgo.Session
	ticker   *time.Ticker
	stopChan chan struct{}
}

// New creates a new Voter instance
func New(session *discordgo.Session) *Voter {
	return &Voter{
		session:  session,
		stopChan: make(chan struct{}),
	}
}

// Start begins the scheduled vote checking process (every 10 minutes)
func (v *Voter) Start() {
	log.Println("Starting vote processor...")
	v.ticker = time.NewTicker(10 * time.Minute)

	// Run immediately on start
	go v.checkExpiredVotes()

	// Then run on schedule
	go func() {
		for {
			select {
			case <-v.ticker.C:
				v.checkExpiredVotes()
			case <-v.stopChan:
				return
			}
		}
	}()

	log.Println("Vote processor started (checking every 10 minutes)")
}

// Stop halts the vote checking process
func (v *Voter) Stop() {
	if v.ticker != nil {
		v.ticker.Stop()
	}
	close(v.stopChan)
	log.Println("Vote processor stopped")
}

// checkExpiredVotes processes all resources with expired voting periods
func (v *Voter) checkExpiredVotes() {
	resources, err := database.GetPendingResourcesWithExpiredVotes()
	if err != nil {
		log.Printf("Error fetching expired votes: %v", err)
		return
	}

	if len(resources) == 0 {
		return
	}

	log.Printf("Processing %d expired vote(s)", len(resources))

	for _, resource := range resources {
		v.processVoteResult(resource)
	}
}

// processVoteResult determines the outcome of a vote and updates the resource
func (v *Voter) processVoteResult(resource database.PublicResource) {
	totalVotes := resource.UsefulVotes + resource.NotUsefulVotes

	var status database.ResourceStatus
	var resultEmbed *discordgo.MessageEmbed

	if totalVotes == 0 {
		// No votes = reject
		status = database.ResourceStatusRejected
		resultEmbed = v.createRejectedEmbed(resource, "No votes received")
	} else if resource.UsefulVotes > resource.NotUsefulVotes {
		// More useful votes = approve
		status = database.ResourceStatusApproved
		resultEmbed = v.createApprovedEmbed(resource)
	} else {
		// Tie or more not useful = reject
		status = database.ResourceStatusRejected
		resultEmbed = v.createRejectedEmbed(resource, "Not enough useful votes")
	}

	// Update status in database
	now := time.Now()
	err := database.UpdateResourceStatus(resource.ID, status, now)
	if err != nil {
		log.Printf("Error updating resource status: %v", err)
		return
	}

	// Send result notification in the same channel as the vote message
	if resource.VoteChannelID != "" {
		_, err = v.session.ChannelMessageSendEmbed(resource.VoteChannelID, resultEmbed)
		if err != nil {
			log.Printf("Error sending vote result: %v", err)
		}
	}

	log.Printf("Processed vote for resource '%s': %s (👍 %d / 👎 %d)",
		resource.Title, status, resource.UsefulVotes, resource.NotUsefulVotes)
}

// createApprovedEmbed creates an embed for an approved resource
func (v *Voter) createApprovedEmbed(resource database.PublicResource) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "✅ Resource Approved!",
		Description: fmt.Sprintf("**%s**\n\nThe community has approved this resource!", resource.Title),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔗 URL",
				Value:  resource.URL,
				Inline: false,
			},
			{
				Name:   "📊 Vote Results",
				Value:  fmt.Sprintf("👍 Useful: **%d**\n👎 Not Useful: **%d**", resource.UsefulVotes, resource.NotUsefulVotes),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Access this resource with /resource list",
		},
	}
}

// createRejectedEmbed creates an embed for a rejected resource
func (v *Voter) createRejectedEmbed(resource database.PublicResource, reason string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "❌ Resource Not Approved",
		Description: fmt.Sprintf("**%s**\n\n%s", resource.Title, reason),
		Color:       0xFF0000, // Red
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📊 Vote Results",
				Value:  fmt.Sprintf("👍 Useful: **%d**\n👎 Not Useful: **%d**", resource.UsefulVotes, resource.NotUsefulVotes),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Feel free to submit a different resource with /resource submit",
		},
	}
}
