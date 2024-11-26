package components

type navigationLink struct {
	Text string
	Href string
}

var navigationLinks = []navigationLink{
	{Text: "Home", Href: "/"},
	{Text: "Services", Href: "#services"},
	{Text: "Instructions", Href: "#instructions"},
}
