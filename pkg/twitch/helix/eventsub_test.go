package helix

// func TestSubscribptions(t *testing.T) {
// 	conf, err := config.Read()
// 	require.NoError(t, err)

// 	c := twitch.NewClient(twitch.WithOAuthCreds(&conf.OAuthCreds))
// 	eventsub := New(c)

// 	ctx := context.Background()

// 	ch, err := c.ChannelRoot_AboutPanel(ctx, "slorpglorpski")
// 	fmt.Println(ch.User.ID)

// 	ev := eventsub.StreamOnlineEvent(ch.User.ID)

// 	b, err := json.MarshalIndent(ev, "", " ")
// 	require.NoError(t, err)

// 	fmt.Println(string(b))

// 	err = eventsub.Subscribe(ctx, ev)
// 	require.NoError(t, err)

// 	err = eventsub.Subscriptions(ctx)
// 	require.NoError(t, err)
// }
