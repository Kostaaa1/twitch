package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/charmbracelet/lipgloss"
)

var (
	colorGreen       = lipgloss.Color("#00D26A")
	colorBlue        = lipgloss.Color("#58A6FF")
	colorPurple      = lipgloss.Color("#8839ef")
	colorMuted       = lipgloss.Color("#8B8B8B")
	greenStyle       = lipgloss.NewStyle().Foreground(colorGreen)
	blueStyle        = lipgloss.NewStyle().Foreground(colorBlue)
	purpleStyle      = lipgloss.NewStyle().Foreground(colorPurple)
	mutedStyle       = lipgloss.NewStyle().Foreground(colorMuted)
	tab              = "  "
	maxContentLength = 600
)

func humanIntFormat(n int) string {
	number := strconv.Itoa(n)
	var s strings.Builder
	for i, r := range number {
		if i > 0 && (len(number)-i)%3 == 0 {
			s.WriteByte(',')
		}
		s.WriteRune(r)
	}
	return s.String()
}

func printLabelRow(s *strings.Builder,
	label,
	value string,
	labelLen,
	maxLen int,
) {
	s.WriteString("\n")
	space := strings.Repeat(" ", maxLen-labelLen+len(tab)*2)
	s.WriteString(tab)
	s.WriteString(tab)
	s.WriteString(label)
	s.WriteString(space)
	s.WriteString(value)
}

func printInfo(s *strings.Builder, u *gql.ChannelRoot_AboutPanel) {
	max := 0
	labels := []string{
		"ID",
		"Followers",
		"Last Game",
		"Subscribers",
		"Date of creation",
	}
	for _, label := range labels {
		if len(label) > max {
			max = len(label)
		}
	}
	style := mutedStyle
	printLabelRow(s, style.Render("ID"), u.User.ID, 2, max)
	printLabelRow(s, style.Render("Followers"), strconv.Itoa(u.User.Followers.TotalCount), len("Followers"), max)
	printLabelRow(s, style.Render("Last Game"), u.User.LastBroadcast.Game.DisplayName, len("Last Game"), max)
	printLabelRow(s, style.Render("Subscribers"), "2000", len("Subscribers"), max)
}

func printHeaderLabel(s *strings.Builder, line string) {
	dash := mutedStyle.Render("──")
	s.WriteString(dash)
	s.WriteString(purpleStyle.
		Bold(true).
		Render(line),
	)
	s.WriteString(strings.Repeat(dash, 20))
}

func printClips(s *strings.Builder, clips *gql.ClipsCardsUser) {
	s.WriteString("\n")

	printHeaderLabel(
		s,
		fmt.Sprintf(" Clips (%d) ", len(clips.User.Clips.Edges)),
	)

	if len(clips.User.Clips.Edges) == 0 {
		s.WriteString(tab)
		s.WriteString(tab)
		s.WriteString(mutedStyle.Render("No videos found"))
		return
	}

	s.WriteString("\n")
	s.WriteString("\n")

	for i, edge := range clips.User.Clips.Edges {
		clip := edge.Node

		numerical := strconv.Itoa(i+1) + "." + " "
		s.WriteString(tab)
		s.WriteString(mutedStyle.Render(numerical))

		if len(clip.Title) > maxContentLength {
			s.WriteString(fmt.Sprintf("%s ...", clip.Title[:50]))
		} else {
			s.WriteString(clip.Title)
		}
		s.WriteString("\n")

		// s.WriteString(tab)
		s.WriteString(strings.Repeat(" ", len(numerical)+2))

		var inner strings.Builder

		// inner.WriteString(clip.)
		// inner.WriteString(" . ")

		inner.WriteString(clip.Game.Name)
		inner.WriteString(" . ")

		inner.WriteString(fmt.Sprintf("%s views", humanIntFormat(clip.ViewCount)))
		inner.WriteString(" . ")

		inner.WriteString(clip.Curator.DisplayName)
		inner.WriteString(" . ")

		seconds := time.Duration(int64(clip.DurationSeconds)) * time.Second
		inner.WriteString(seconds.String())
		inner.WriteString(" . ")

		since := time.Since(clip.CreatedAt)
		_ = since
		inner.WriteString("7y" + " ago")

		s.WriteString(mutedStyle.Render(inner.String()))
		s.WriteString("\n")
	}
}

func printSocials(s *strings.Builder, u *gql.ChannelRoot_AboutPanel) {
	s.WriteString("\n\n")
	s.WriteString(tab)
	s.WriteString(
		purpleStyle.
			Bold(true).
			Render("Socials"),
	)
	s.WriteString("\n")

	maxSpace := 0
	for _, social := range u.User.Channel.SocialMedias {
		if len(social.Name) > maxSpace {
			maxSpace = len(social.Name)
		}
	}
	for _, social := range u.User.Channel.SocialMedias {
		printLabelRow(
			s,
			mutedStyle.Render(social.Name),
			blueStyle.Render(social.URL),
			len(social.Name),
			maxSpace,
		)
	}
}

func printAbout(s *strings.Builder, u *gql.ChannelRoot_AboutPanel) {
	s.WriteString("\n\n")
	s.WriteString(tab)
	s.WriteString(
		purpleStyle.
			Bold(true).
			Render("About"),
	)
	s.WriteString("\n")
	s.WriteString("\n")
	s.WriteString(tab)
	s.WriteString(tab)
	s.WriteString(
		mutedStyle.
			Render(u.User.Description),
	)
}

func printChannel(s *strings.Builder, u *gql.ChannelRoot_AboutPanel, userStyle lipgloss.Style) {
	checkMark := greenStyle.Bold(true).Render("✓")

	var inner strings.Builder

	if u.User.Roles.IsPartner {
		inner.WriteString(mutedStyle.Render("Partnered "))
		inner.WriteString(checkMark)
	}

	if u.User.Roles.IsAffiliate {
		inner.WriteString(mutedStyle.Render("Afiliated "))
		inner.WriteString(checkMark)
	}

	header := userStyle.
		Border(lipgloss.RoundedBorder()).
		UnsetBorderBackground().
		PaddingLeft(2).
		PaddingRight(2).
		BorderForeground(colorPurple).
		Render(fmt.Sprintf("%s %s", u.User.DisplayName, inner.String()))

	s.WriteString(header)
}

func printVideos(s *strings.Builder, videos *gql.FilterableVideoTower_Videos) {
	s.WriteString("\n")
	s.WriteString("\n")

	line := fmt.Sprintf(" Videos (%d) ", len(videos.User.Videos.Edges))
	printHeaderLabel(s, line)

	s.WriteString("\n")
	s.WriteString("\n")

	if len(videos.User.Videos.Edges) == 0 {
		s.WriteString(tab)
		s.WriteString(tab)
		s.WriteString(mutedStyle.Render("No videos found"))
		return
	}

	for i, edge := range videos.User.Videos.Edges {
		video := edge.Node

		numerical := strconv.Itoa(i+1) + "." + " "
		s.WriteString(tab)
		s.WriteString(mutedStyle.Render(numerical))

		if len(video.Title) > maxContentLength {
			s.WriteString(fmt.Sprintf("%s ...", video.Title[:50]))
		} else {
			s.WriteString(video.Title)
		}
		s.WriteString("\n")

		// s.WriteString(tab)
		s.WriteString(strings.Repeat(" ", len(numerical)+2))

		var inner strings.Builder

		inner.WriteString(video.Game.DisplayName)
		inner.WriteString(" . ")

		inner.WriteString(video.ID)
		inner.WriteString(" . ")

		seconds := time.Duration(int64(video.LengthSeconds)) * time.Second
		inner.WriteString(seconds.String())
		inner.WriteString(" . ")

		strconv.Itoa(video.ViewCount)
		inner.WriteString(fmt.Sprintf("%s views", humanIntFormat(video.ViewCount)))
		inner.WriteString(" . ")

		since := time.Since(video.PublishedAt)
		_ = since
		inner.WriteString("7y" + " ago")

		s.WriteString(mutedStyle.Render(inner.String()))

		s.WriteString("\n")
	}
}

func PrintChannel(
	about *gql.ChannelRoot_AboutPanel,
	videos *gql.FilterableVideoTower_Videos,
	clips *gql.ClipsCardsUser,
) {
	u := about.User

	if u.PrimaryColorHex == "" {
		u.PrimaryColorHex = "FFBF00"
	}

	primaryHex := fmt.Sprintf("#%s", u.PrimaryColorHex)
	colorUserPrimary := lipgloss.Color(primaryHex)
	userStyle := lipgloss.NewStyle().Foreground(colorUserPrimary)

	var s strings.Builder
	// Header
	printChannel(&s, about, userStyle)
	// Info
	printInfo(&s, about)
	// About
	printAbout(&s, about)
	// Socials
	printSocials(&s, about)
	//videso
	printVideos(&s, videos)
	//videso
	printClips(&s, clips)

	s.WriteString("\n")

	fmt.Println(s.String())
}
