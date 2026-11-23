package main

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// customer is a customer of the shop, in a region.
type customer struct {
	ID                       int
	Code, Name, City, Region string
	Orders                   int
}

// customers are the shop's six customers, in two regions.
var customers = []customer{
	{1, "C001", "Acme", "Hanoi", "North", 12},
	{2, "C002", "Birch", "Haiphong", "North", 5},
	{3, "C003", "Cedar", "Hue", "North", 8},
	{4, "C004", "Delta", "Saigon", "South", 21},
	{5, "C005", "Elm", "Can Tho", "South", 3},
	{6, "C006", "Fern", "Da Lat", "South", 7},
}

// regions are the regions of customers, in the order shown.
var regions = []string{"North", "South"}

// node is a row of the customers tree: a region or a customer.
type node struct {
	Key, Name, City string
	Orders          int
}

func (n *node) orders() string { return strconv.Itoa(n.Orders) }

// roots are the regions of the tree, each with the orders of its customers,
// and kids the customers of a region.
func roots() []node {
	var out []node
	for _, r := range regions {
		n := node{Key: "region:" + r, Name: r}
		for _, c := range customers {
			if c.Region == r {
				n.Orders += c.Orders
			}
		}
		out = append(out, n)
	}
	return out
}

func kids(region string) []node {
	var out []node
	for _, c := range customers {
		if c.Region == region {
			out = append(out, node{Key: "customer:" + strconv.Itoa(c.ID), Name: c.Name, City: c.City, Orders: c.Orders})
		}
	}
	return out
}

// icon draws Lucide shapes (ISC license), 24 units wide, as ui/widget's own
// icons are drawn.
func icon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

// The shapes of Lucide's file-plus, save, x, menu, info, clipboard-list,
// users and palette.
const (
	shapesNew      = `<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M9 15h6"/><path d="M12 18v-6"/>`
	shapesSave     = `<path d="M15.2 3a2 2 0 0 1 1.4.6l3.8 3.8a2 2 0 0 1 .6 1.4V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z"/><path d="M17 21v-7a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1v7"/><path d="M7 3v4a1 1 0 0 0 1 1h7"/>`
	shapesClose    = `<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`
	shapesMenu     = `<path d="M3 12h18"/><path d="M3 6h18"/><path d="M3 18h18"/>`
	shapesInfo     = `<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>`
	shapesOrders   = `<rect width="8" height="4" x="8" y="2" rx="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><path d="M12 11h4"/><path d="M12 16h4"/><path d="M8 11h.01"/><path d="M8 16h.01"/>`
	shapesCustomer = `<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>`
	shapesSkin     = `<circle cx="13.5" cy="6.5" r=".5"/><circle cx="17.5" cy="10.5" r=".5"/><circle cx="8.5" cy="7.5" r=".5"/><circle cx="6.5" cy="12.5" r=".5"/><path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.926 0 1.648-.746 1.648-1.688 0-.437-.18-.835-.437-1.125-.29-.289-.438-.652-.438-1.125a1.64 1.64 0 0 1 1.668-1.668h1.996c3.051 0 5.555-2.503 5.555-5.554C21.965 6.012 17.461 2 12 2z"/>`
)
