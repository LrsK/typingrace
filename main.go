package main

import (
	"bufio"
	"flag"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

type word struct {
	index int
	text  string
	lead  int
	row   int
	color termbox.Attribute
}

var (
	dictionary     = []string{}
	words          = []word{}
	input          = make([]rune, 0, 64)
	count          = '0'
	pos            = 0
	points         = 0
	wordChan       = make(chan string, 2)
	pause          = make(chan bool)
	resume         = make(chan bool)
	width, height  int
	wordLock       sync.Mutex
	r              *rand.Rand
	difficultyFlag = flag.String("d", "medium", "How fast the words are moving and generated.")
)

const ColDef = termbox.ColorDefault
const ColBlk = termbox.ColorBlack
const ColRed = termbox.ColorRed
const ColGreen = termbox.ColorGreen

func main() {
	s := rand.NewSource(time.Now().UnixNano())
	r = rand.New(s)

	loadDictionary()

	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	width, height = termbox.Size()

	// Listen for events
	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()

	difficulty := *difficultyFlag

	var drawTicker <-chan time.Time
	fastDrawTicker := time.Tick(100 * time.Millisecond)
	mediumDrawTicker := time.Tick(500 * time.Millisecond)
	slowDrawTicker := time.Tick(1000 * time.Millisecond)

	var wordTicker <-chan time.Time
	fastWordTicker := time.Tick(1 * time.Second)
	mediumWordTicker := time.Tick(3 * time.Second)
	slowWordTicker := time.Tick(6 * time.Second)

	if difficulty == "easy" {
		wordTicker = slowWordTicker
		drawTicker = slowDrawTicker
	} else if difficulty == "medium" {
		wordTicker = mediumWordTicker
		drawTicker = mediumDrawTicker
	} else if difficulty == "hard" {
		wordTicker = fastWordTicker
		drawTicker = fastDrawTicker
	} else {
		wordTicker = fastWordTicker
		drawTicker = fastDrawTicker
	}

	// Fire new words at given intervals
	go func() {
		for {
			select {
			case <-pause:
				<-resume
			case <-wordTicker:
				wordLock.Lock()
				addWord()
				wordLock.Unlock()
			}
		}
	}()

	code := 0

loop:
	for {
		select {
		case ev := <-eventQueue:
			if ev.Type == termbox.EventKey && ev.Key == termbox.KeyEsc {
				break loop
			} else if ev.Type == termbox.EventKey &&
				(ev.Key == termbox.KeyEnter) && len(input) > 0 {
				input = []rune{}
			} else if ev.Type == termbox.EventKey &&
				(ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2) && len(input) > 0 {
				input = input[:len(input)-1]
			} else {
				input = append(input, ev.Ch)
			}
		case <-drawTicker:
			wordLock.Lock()
			code = draw()
			if code == -1 {
				break loop
			}
			wordLock.Unlock()
		}

		// Check for
		wordLock.Lock()
		for _, w := range words {
			if w.text == string(input) {
				removeWord(w)
				input = []rune{}
				points += (100 - w.lead%100) + len(w.text)
			}
		}
		wordLock.Unlock()
	}

	if code == -1 {
		time.Sleep(4 * time.Second)
	}

}

func draw() int {
	termbox.Clear(ColGreen, ColDef)
	x, y := 0, 0
	code := 0

	// Draw moving words
	for _, w := range words {
		for j, c := range w.text {
			termbox.SetCell(x+w.lead+j, y+w.row, c, w.color, ColDef)
		}
		words[w.index].lead++

		if words[w.index].lead > width*75/100 {
			words[w.index].color = ColRed
		}

		// End game
		if w.lead > width {
			ps := strconv.Itoa(points)
			ps = "Your score: " + ps + ". Press ESC to exit."
			pr := []rune(ps)
			for i, r := range pr {
				termbox.SetCell((width/2-len(ps)/2)+i, height/2, r, ColGreen, ColDef)
			}
			code = 1
			break
		}
	}

	// Print input for debug
	/*
		for i, v := range input {
			termbox.SetCell(width/2+i, height/2, v, ColDef, ColDef)
		}
	*/
	termbox.Flush()
	return code
}

func addWord() {
	words = append(words, word{len(words), dictionary[r.Intn(len(dictionary))], 0, r.Intn(height), ColDef})
}

func removeWord(w word) {
	if w.index == 0 {
		words = words[1:]
		for i, _ := range words {
			words[i].index = i
		}
	} else {
		words = words[:w.index+copy(words[w.index:], words[w.index+1:])]
		for i, _ := range words {
			words[i].index = i
		}
	}
}

func loadDictionary() {
	file, err := os.Open("words")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	c := 0
	for scanner.Scan() {
		if c >= 2000 {
			break
		} else if r.Intn(100) < 30 {
			dictionary = append(dictionary, scanner.Text())
		}

	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}
