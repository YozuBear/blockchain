package art_app_tests

func GetLongestChainTest(num int) (genHash string, expected []string) {
	switch num {
	case 1:
		// 1st tree structure for testing longest chain:
		//     A
		//    / \
		//   B   C
		//  / \   \
		// D   E   H
		// |   |
		// F   I
		// |
		// G
		// Expected longest chain: G F D B A
		genHash = "A"
		expected = []string{"G", "F", "D", "B", "A"}
	case 2:
		// 2nd tree structure for testing longest chain:
		//         Z
		//       / | \
		//     O   P   J
		//   / |   |    \
		//  R  S   Q     K
		//  |  |         |
		//  T  U         L
		//               |
		//               M
		//               |
		//               N
		//
		//
		//  Expected longest chain： [N M L K J Z]
		genHash = "Z"
		expected = []string{"N", "M", "L", "K", "J", "Z"}
	case 3:
		// 3rd tree structure for testing longest chain
		//          й
		//        / | \
		//       ц  у  к
		//      /  / \  | \
		//     е  н   г ш  з
		//　　    / | \   \  \
		//     ф  ы  в   а  п
		//        |   \      \
		//		  р    о      д
		//        |
		//        я
		// Expected longest chain: я р ы н у й
		genHash = "й"
		expected = []string{"я", "р", "ы", "н", "у", "й"}
	default:
		// same as case 1
		genHash = "A"
		expected = []string{"G", "F", "D", "B", "A"}
	}
	return
}

func GetChildren(hash string) (children []string) {

	switch hash {
	// case 1
	case "A":
		children = []string{"B", "C"}
	case "B":
		children = []string{"D", "E"}
	case "C":
		children = []string{"H"}
	case "D":
		children = []string{"F"}
	case "F":
		children = []string{"G"}
	case "E":
		children = []string{"I"}
	// case 2
	case "Z":
		children = []string{"O", "P", "J"}
	case "O":
		children = []string{"R", "S"}
	case "P":
		children = []string{"Q"}
	case "J":
		children = []string{"K"}
	case "R":
		children = []string{"T"}
	case "S":
		children = []string{"U"}
	case "K":
		children = []string{"L"}
	case "L":
		children = []string{"M"}
	case "M":
		children = []string{"N"}

	// case 3
	case "й":
		children = []string{"ц", "у", "к"}
	case "ц":
		children = []string{"е"}
	case "у":
		children = []string{"н", "г"}
	case "к":
		children = []string{"ш", "з"}
	case "н":
		children = []string{"ф", "ы", "в"}
	case "ш":
		children = []string{"а"}
	case "з":
		children = []string{"п"}
	case "ы":
		children = []string{"р"}
	case "в":
		children = []string{"о"}
	case "п":
		children = []string{"д"}
	case "р":
		children = []string{"я"}

	default:
		children = []string{}
	}
	return
}
