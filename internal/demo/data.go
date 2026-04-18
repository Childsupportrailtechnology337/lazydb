package demo

// FirstNames contains 100 realistic first names.
var FirstNames = []string{
	"James", "Mary", "Robert", "Patricia", "John", "Jennifer", "Michael", "Linda",
	"David", "Elizabeth", "William", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Christopher", "Karen", "Charles", "Lisa", "Daniel", "Nancy",
	"Matthew", "Betty", "Anthony", "Margaret", "Mark", "Sandra", "Donald", "Ashley",
	"Steven", "Dorothy", "Paul", "Kimberly", "Andrew", "Emily", "Joshua", "Donna",
	"Kenneth", "Michelle", "Kevin", "Carol", "Brian", "Amanda", "George", "Melissa",
	"Timothy", "Deborah", "Ronald", "Stephanie", "Edward", "Rebecca", "Jason", "Sharon",
	"Jeffrey", "Laura", "Ryan", "Cynthia", "Jacob", "Kathleen", "Gary", "Amy",
	"Nicholas", "Angela", "Eric", "Shirley", "Jonathan", "Anna", "Stephen", "Brenda",
	"Larry", "Pamela", "Justin", "Emma", "Scott", "Nicole", "Brandon", "Helen",
	"Benjamin", "Samantha", "Samuel", "Katherine", "Raymond", "Christine", "Gregory", "Debra",
	"Frank", "Rachel", "Alexander", "Carolyn", "Patrick", "Janet", "Jack", "Catherine",
	"Dennis", "Maria", "Jerry", "Heather",
}

// LastNames contains 80 realistic last names.
var LastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas",
	"Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson", "White",
	"Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson", "Walker", "Young",
	"Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
	"Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell",
	"Carter", "Roberts", "Gomez", "Phillips", "Evans", "Turner", "Diaz", "Parker",
	"Cruz", "Edwards", "Collins", "Reyes", "Stewart", "Morris", "Morales", "Murphy",
	"Cook", "Rogers", "Gutierrez", "Ortiz", "Morgan", "Cooper", "Peterson", "Bailey",
	"Reed", "Kelly", "Howard", "Ramos", "Kim", "Cox", "Ward", "Richardson",
}

// ProductData holds a product name and its category.
type ProductData struct {
	Name     string
	Category string
}

// Products contains 50 products with their categories.
var Products = []ProductData{
	// Electronics
	{"Wireless Bluetooth Headphones", "Electronics"},
	{"USB-C Charging Cable", "Electronics"},
	{"Portable Power Bank 10000mAh", "Electronics"},
	{"Mechanical Keyboard RGB", "Electronics"},
	{"Wireless Mouse Ergonomic", "Electronics"},
	{"4K Webcam with Microphone", "Electronics"},
	{"Noise Cancelling Earbuds", "Electronics"},
	{"Smart Watch Fitness Tracker", "Electronics"},
	{"Portable SSD 1TB", "Electronics"},
	{"LED Desk Lamp Adjustable", "Electronics"},
	// Clothing
	{"Cotton Crew Neck T-Shirt", "Clothing"},
	{"Slim Fit Denim Jeans", "Clothing"},
	{"Waterproof Winter Jacket", "Clothing"},
	{"Running Shoes Lightweight", "Clothing"},
	{"Merino Wool Socks Pack", "Clothing"},
	{"Leather Belt Classic", "Clothing"},
	{"Polarized Sunglasses UV400", "Clothing"},
	{"Baseball Cap Adjustable", "Clothing"},
	// Home & Kitchen
	{"Stainless Steel Water Bottle", "Home & Kitchen"},
	{"Non-Stick Frying Pan 12in", "Home & Kitchen"},
	{"French Press Coffee Maker", "Home & Kitchen"},
	{"Bamboo Cutting Board Set", "Home & Kitchen"},
	{"Silicone Spatula Set", "Home & Kitchen"},
	{"Insulated Travel Mug", "Home & Kitchen"},
	{"Cast Iron Dutch Oven", "Home & Kitchen"},
	{"Electric Kettle 1.7L", "Home & Kitchen"},
	// Books
	{"Introduction to Algorithms", "Books"},
	{"Design Patterns in Go", "Books"},
	{"The Pragmatic Programmer", "Books"},
	{"Clean Code Handbook", "Books"},
	{"Database Internals Guide", "Books"},
	{"Distributed Systems Concepts", "Books"},
	// Sports & Outdoors
	{"Yoga Mat Non-Slip", "Sports & Outdoors"},
	{"Resistance Bands Set", "Sports & Outdoors"},
	{"Adjustable Dumbbells 25lb", "Sports & Outdoors"},
	{"Camping Tent 4-Person", "Sports & Outdoors"},
	{"Hiking Backpack 40L", "Sports & Outdoors"},
	{"Jump Rope Speed Cable", "Sports & Outdoors"},
	// Office Supplies
	{"Gel Pen Set 12-Pack", "Office Supplies"},
	{"A5 Notebook Hardcover", "Office Supplies"},
	{"Desk Organizer Wooden", "Office Supplies"},
	{"Whiteboard Markers Set", "Office Supplies"},
	{"Document Scanner Portable", "Office Supplies"},
	// Health & Beauty
	{"Vitamin D3 Supplements", "Health & Beauty"},
	{"Organic Lip Balm Pack", "Health & Beauty"},
	{"Electric Toothbrush Sonic", "Health & Beauty"},
	{"Hand Cream Moisturizing", "Health & Beauty"},
	// Toys & Games
	{"Strategy Board Game Classic", "Toys & Games"},
	{"1000-Piece Jigsaw Puzzle", "Toys & Games"},
	{"Building Blocks Set 500pc", "Toys & Games"},
}

// Categories lists the distinct product categories.
var Categories = []string{
	"Electronics",
	"Clothing",
	"Home & Kitchen",
	"Books",
	"Sports & Outdoors",
	"Office Supplies",
	"Health & Beauty",
	"Toys & Games",
	"Automotive",
	"Garden & Patio",
}

// Cities contains 30 city names.
var Cities = []string{
	"New York", "Los Angeles", "Chicago", "Houston", "Phoenix",
	"Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose",
	"Austin", "Jacksonville", "Fort Worth", "Columbus", "Charlotte",
	"Indianapolis", "San Francisco", "Seattle", "Denver", "Nashville",
	"Portland", "Oklahoma City", "Las Vegas", "Memphis", "Louisville",
	"Baltimore", "Milwaukee", "Albuquerque", "Tucson", "Mesa",
}

// States corresponding to the cities above (index-matched).
var States = []string{
	"NY", "CA", "IL", "TX", "AZ",
	"PA", "TX", "CA", "TX", "CA",
	"TX", "FL", "TX", "OH", "NC",
	"IN", "CA", "WA", "CO", "TN",
	"OR", "OK", "NV", "TN", "KY",
	"MD", "WI", "NM", "AZ", "AZ",
}

// EmailDomains for generating realistic email addresses.
var EmailDomains = []string{
	"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "icloud.com",
	"protonmail.com", "fastmail.com", "zoho.com", "aol.com", "mail.com",
}

// ReviewTemplates are sentence templates for product reviews.
// The placeholder {product} can be replaced with the product name.
var ReviewTemplates = []string{
	"Absolutely love this product! Works exactly as described.",
	"Great quality for the price. Would definitely recommend.",
	"Decent product but took a while to arrive.",
	"Not what I expected. The quality could be better.",
	"Five stars! This exceeded my expectations in every way.",
	"Good value for money. Solid build quality.",
	"The product arrived damaged. Customer service was helpful though.",
	"I've been using this for months and it still works perfectly.",
	"Bought this as a gift and the recipient loved it.",
	"Average product. Does the job but nothing special.",
	"Incredible quality. I'm ordering another one for my friend.",
	"Shipping was fast and the packaging was excellent.",
	"A bit overpriced for what you get, but still decent.",
	"This is exactly what I needed. Perfect fit for my setup.",
	"Stopped working after two weeks. Very disappointed.",
	"Best purchase I've made this year. Highly recommend!",
	"The color was slightly different from the picture but still nice.",
	"Easy to set up and use. Great for beginners.",
	"Premium feel and finish. You can tell this is well-made.",
	"Works as advertised. No complaints whatsoever.",
	"Returned this because it didn't match the description.",
	"Outstanding product! Will buy from this seller again.",
	"The instructions were unclear but the product itself is fine.",
	"Surprisingly good quality. Better than more expensive alternatives.",
	"My second time buying this. Just as good as the first.",
	"Functional but the design could use some improvement.",
	"Perfect for everyday use. Very practical and durable.",
	"Took a star off because the packaging was poor.",
	"This has become an essential part of my daily routine.",
	"Would give six stars if I could. Absolutely fantastic!",
}

// StreetNames for generating addresses.
var StreetNames = []string{
	"Main St", "Oak Ave", "Maple Dr", "Cedar Ln", "Elm St",
	"Pine Rd", "Washington Blvd", "Park Ave", "Lake Dr", "Hill St",
	"River Rd", "Sunset Blvd", "Broadway", "Highland Ave", "Forest Dr",
	"Meadow Ln", "Spring St", "Valley Rd", "Church St", "Market St",
}

// OrderStatuses for the orders table.
var OrderStatuses = []string{
	"pending", "processing", "shipped", "delivered", "cancelled", "returned",
}
