package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/pkg/twitch/gql"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
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
	checkMark        = greenStyle.Bold(true).Render("✓")
	tab              = "  "
	maxContentLength = 600
	width            = 80
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
	space := strings.Repeat(" ", maxLen-labelLen+len(tab)*2)
	s.WriteString(tab)
	s.WriteString(label)
	s.WriteString(space)
	s.WriteString(value)
	s.WriteString("\n")
}

func printHeaderLabel(s *strings.Builder, label string) {
	s.WriteString(tab)
	dash := mutedStyle.Render("──")
	s.WriteString(dash)
	s.WriteString(purpleStyle.Bold(true).Render(fmt.Sprintf(" %s ", label)))
	s.WriteString(strings.Repeat(dash, (width-len(label))/2-5))
	s.WriteString("\n\n")
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
	s.WriteString("\n")
}

func printClips(s *strings.Builder, clips *gql.ClipsCardsUser) {
	printHeaderLabel(s, fmt.Sprintf("Clips (%d)", len(clips.User.Clips.Edges)))

	if len(clips.User.Clips.Edges) == 0 {
		s.WriteString(tab)
		s.WriteString(mutedStyle.Render("No clips found"))
	} else {
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

			var inner strings.Builder
			inner.WriteString(strings.Repeat(" ", len(numerical)+2))
			inner.WriteString(clip.Game.Name)
			inner.WriteString(" . ")
			inner.WriteString(clip.Slug)
			inner.WriteString(" . ")
			inner.WriteString(fmt.Sprintf("%s views", humanIntFormat(clip.ViewCount)))
			inner.WriteString(" . ")
			inner.WriteString(clip.Curator.DisplayName)
			inner.WriteString(" . ")
			seconds := time.Duration(int64(clip.DurationSeconds)) * time.Second
			inner.WriteString(seconds.String())
			inner.WriteString(" . ")
			// since := time.Since(clip.CreatedAt) USE FOR AGO
			inner.WriteString("7y" + " ago")

			s.WriteString("\n")
			s.WriteString(mutedStyle.Render(inner.String()))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
}

func printSocials(s *strings.Builder, u *gql.ChannelRoot_AboutPanel) {
	printHeaderLabel(s, "Socials")

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
	s.WriteString("\n")
}

func printAbout(s *strings.Builder, u *gql.ChannelRoot_AboutPanel) {
	printHeaderLabel(s, "About")
	s.WriteString(tab)
	s.WriteString(mutedStyle.Render(u.User.Description))
	s.WriteString("\n\n")
}

func printChannel(s *strings.Builder, u *gql.ChannelRoot_AboutPanel, userStyle lipgloss.Style) {
	values := make([]string, 0)
	if u.User.Roles.IsPartner {
		values = append(values, mutedStyle.Render("Partnered ")+checkMark)
	}
	if u.User.Roles.IsAffiliate {
		values = append(values, mutedStyle.Render("Afiliated ")+checkMark)
	}

	var inner strings.Builder
	inner.WriteString(strings.Join(values, " "))

	header := userStyle.
		Border(lipgloss.RoundedBorder()).
		UnsetBorderBackground().
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(colorPurple).
		Render(fmt.Sprintf("%s %s", u.User.DisplayName, inner.String()))

	s.WriteString(header)
	s.WriteString("\n")
}

func printVideos(s *strings.Builder, videos *gql.FilterableVideoTower_Videos) {
	printHeaderLabel(s, fmt.Sprintf("Videos (%d)", len(videos.User.Videos.Edges)))

	if len(videos.User.Videos.Edges) == 0 {
		s.WriteString(tab)
		s.WriteString(tab)
		s.WriteString(mutedStyle.Render("No videos found"))
	} else {

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

			var inner strings.Builder

			inner.WriteString(strings.Repeat(" ", len(numerical)+2))

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

			s.WriteString("\n")
			s.WriteString(mutedStyle.Render(inner.String()))
			s.WriteString("\n")
		}
	}
	s.WriteString("\n")
}

func getTerminalWidth() (int, int, error) {
	fd := uintptr(os.Stdout.Fd())
	return term.GetSize(fd)
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

	w, _, _ := getTerminalWidth()
	if w > 0 {
		width = w
	}

	primaryHex := fmt.Sprintf("#%s", u.PrimaryColorHex)
	colorUserPrimary := lipgloss.Color(primaryHex)
	userStyle := lipgloss.NewStyle().Foreground(colorUserPrimary)

	var s strings.Builder
	printChannel(&s, about, userStyle)
	printInfo(&s, about)
	printAbout(&s, about)
	printSocials(&s, about)
	printVideos(&s, videos)
	printClips(&s, clips)
	fmt.Println(s.String())
}
