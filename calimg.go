package blackout

import (
	"cloud.google.com/go/storage"
	"github.com/tortuoise/wattwerks/utils"
	"golang.org/x/net/context"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/appengine/file"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"math"
	"os"
	"strconv"
	"time"
)

var (
	//bucket = "blackoutmap.appspot.com"
	months = map[time.Month]string{9: "sep.jpg", 10: "oct.jpg", 11: "nov.jpg"}
)

// CalImage encapsulates the data required to generate calendar images
type CalImage struct {
	ven    *time.Location
	start  time.Time
	end    time.Time
	now    time.Time
	images []*image.Gray
	cols   int
	rows   int
	n35    [][]int // stores the start and end (int 0<n<=42) on grid of each month.
	vnts   *[]*calendar.Event
}

// NewCalImage returns a new CalImage ptr with start and end dates and number of rows and cols in the subimage
func NewCalImage(st, nd time.Time, rs, cs int) *CalImage {

	x := newCalImage(st, nd, rs, cs)
	for i, ov := range x.Overflow() {
		xst := x.start.Month()
		xsti := xst + time.Month(i)
		if ov != 0 {
			x.n35[i] = append(x.n35[i], ov)
			x.n35[i] = append(x.n35[i], x.n35[i][0]+utils.DayspMonth(xsti)) // - 1)
		} else {
			xs := make([]int, 2)
			xs[0] = int(time.Date(x.start.Year(), xsti, 1, 12, 0, 0, 0, x.ven).Weekday()) //+ 1
			xs[1] = xs[0] + utils.DayspMonth(xsti)                                        //- 1
			x.n35[i] = xs                                                                 //append(x.n35[i], int(time.Date(x.start.Year(), x.start.Month(), 1, 12, 0, 0, 0, x.ven).Weekday()))
			//x.n35[i] = append(x.n35[i], x.n35[i][0] + utils.DayspMonth(x.start.Month()))
		}
	}
	return x

}

func newCalImage(st, nd time.Time, rs, cs int) *CalImage {

	venue, _ := time.LoadLocation("UTC")
	if st.Year() == nd.Year() {
		per := nd.Month() - st.Month() + 1
		return &CalImage{ven: venue, start: st, end: nd, cols: cs, rows: rs, n35: make([][]int, per, per), images: make([]*image.Gray, per, per)}
	} else {
		return &CalImage{ven: venue, start: st, end: nd, cols: cs, rows: rs}
	}

}

//CreateImages calls on CalImage.CreateImages & CalImage.WriteCloudImage to create and store monthly images between time.Now().Date() and 2/11/2016
func CreateImages(ctx context.Context) error {

	//grid := 42
	rows := 6
	cols := 7
	now := time.Now() // today's date
	ven, err := time.LoadLocation("IST")
	if err != nil {
		ven, _ = time.LoadLocation("UTC")
	}
	dday := time.Date(2016, time.November, 02, 8, 30, 0, 0, ven)

	ds := 0
	for ds = 0; now.Before(dday); ds++ {
		now = now.Add(time.Hour * 24)
	}

	ci := NewCalImage(time.Now(), dday, rows, cols)
	err = ci.CreateImages()
	if err != nil {
		log.Printf("error creating images: %v\n", err)
		return err
	}
	for i, img := range ci.images {
		err = WriteCloudImage(ctx, img, "f"+strconv.Itoa(i)+".jpg")
		if err != nil {
			log.Printf("error writing images: %v\n", err)
			return err
		}
	}
	return err

}

//CreateImages creates images for all the months between start and end
func (ci *CalImage) CreateImages() error {

	rdr, err := os.Open("W.jpg")
	if err != nil {
		return err
	}
	digit, _, err := image.Decode(rdr)
	if err != nil {
		return err
	}
	rdr.Close()
	digitw := utils.Convert(digit, color.GrayModel.Convert)
	bl := digitw.Bounds()

	mult := bl.Max.Y
	for ii, n35 := range ci.n35 {
		rows := ci.rows
		cols := ci.cols
		grid := rows * cols
		if n35[0] < 0 {
			//rows++
			//grid = rows * cols
			n35[1] = n35[1] - n35[0]
			n35[0] += 7
		}
		// find current month image
		mrdr, err := os.Open(months[ci.start.Month()+time.Month(ii)])
		if err != nil {
			return err
		}
		monthname, _, err := image.Decode(mrdr)
		if err != nil {
			return err
		}
		mrdr.Close()
		monthnameg := utils.Convert(monthname, color.GrayModel.Convert)
		// end current month image
		x, y, p := 0, 0, 1
		days := make([]*image.Gray, 0, grid)
		mth := image.NewGray(image.Rect(0, 0, cols*mult, rows*mult))
		for n := 1; n <= grid; n++ { //n 1...35/42

			//digitgg := utils.Copy(digitg)
			if t := (n % cols); t != 0 {
				x = (n % cols) * mult
			} else {
				x = cols * mult
			}
			if t := (n / cols); n%cols != 0 {
				y = (t + 1) * mult
			} else {
				y = t * mult
			}
			day := mth.SubImage(image.Rect(x-mult, y-mult, x, y)).(*image.Gray)
			days = append(days, day)
			if n35[0] > 0 && n > n35[0] && p <= utils.DayspMonth(ci.start.Month()+time.Month(ii)) {
				rdr, err := os.Open(strconv.Itoa(p) + ".jpg")
				if err != nil {
					return err
				}
				digit, _, err := image.Decode(rdr)
				if err != nil {
					return err
				}
				rdr.Close()
				digitg := utils.Convert(digit, color.GrayModel.Convert)
				digitgg := utils.Copy(digitg)
				grey := color.Gray{127}
				black := color.Gray{0}
				var mark bool
				if (time.Now().Month() == (ci.start.Month()+time.Month(ii)) && time.Now().Day() == p) || (ci.start.Month()+time.Month(ii) == ci.end.Month() && ci.end.Day() == p) {
					mark = true
				}
				for yy := bl.Min.Y; yy < bl.Max.Y; yy++ {
					for xx := bl.Min.X; xx < bl.Max.X; xx++ {
						if gy := digitgg.At(xx, yy).(color.Gray).Y; uint16(gy) > 100 { //!= color.White.Y {
							//digitg.SetGray(xx, yy, grey)
							digitgg.SetGray(xx, yy, grey)
						}
						if mark && (xx < 5 || xx > 45 || yy < 5 || yy > 45) {
							digitgg.SetGray(xx, yy, black)
						}
					}
				}
				draw.Draw(day, day.Bounds(), digitgg, image.ZP, draw.Src)
				p++
			} else {
				draw.Draw(day, day.Bounds(), &image.Uniform{color.Gray{127}}, image.ZP, draw.Src)
				if n == 39 { // mark Wednesday
					draw.Draw(day, day.Bounds(), digitw, image.ZP, draw.Src)

				}
				if n == 42 { // mark month
					grey := color.Gray{127}
					for yy := bl.Min.Y; yy < bl.Max.Y; yy++ {
						for xx := bl.Min.X; xx < bl.Max.X; xx++ {
							if gy := monthnameg.At(xx, yy).(color.Gray).Y; uint16(gy) > 40 { //!= color.White.Y {
								monthnameg.SetGray(xx, yy, grey)
							}
						}
					}
					draw.Draw(day, day.Bounds(), monthnameg, image.ZP, draw.Src)
				}
			}
		}
		//return mth, nil
		//ci.images = append(ci.images, mth)
		ci.images[ii] = mth
	}
	return nil
	//draw.Draw(mth, mth.Bounds(), &image.Uniform{grey}, image.ZP, draw.Src)

}

//ColorImages creates images for all the months between start and end
func (ci *CalImage) ColorImages() error {

	rdr, err := os.Open("W.jpg")
	if err != nil {
		return err
	}
	digit, _, err := image.Decode(rdr)
	if err != nil {
		return err
	}
	rdr.Close()
	digitw := utils.Convert(digit, color.GrayModel.Convert)
	bl := digitw.Bounds()

	mult := bl.Max.Y
	for ii, n35 := range ci.n35 {
		rows := ci.rows
		cols := ci.cols
		grid := rows * cols
		if n35[0] < 0 {
			//rows++
			//grid = rows * cols
			n35[1] = n35[1] - n35[0]
			n35[0] += 7
		}
		crnt_mnth := (ci.start.Month() + time.Month(ii)) % 12
		var crnt_year int
		if crnt_mnth >= ci.start.Month() {
			crnt_year = ci.start.Year()
		} else {
			crnt_year = ci.start.Year() + 1
		}
		// find current month image
		mrdr, err := os.Open(months[crnt_mnth])
		if err != nil {
			return err
		}
		monthname, _, err := image.Decode(mrdr)
		if err != nil {
			return err
		}
		mrdr.Close()
		monthnameg := utils.Convert(monthname, color.GrayModel.Convert)
		// end current month image
		x, y, p := 0, 0, 1
		days := make([]*image.Gray, 0, grid)
		mth := image.NewGray(image.Rect(0, 0, cols*mult, rows*mult))
		for n := 1; n <= grid; n++ { //n 1...35/42

			//digitgg := utils.Copy(digitg)
			if t := (n % cols); t != 0 {
				x = (n % cols) * mult
			} else {
				x = cols * mult
			}
			if t := (n / cols); n%cols != 0 {
				y = (t + 1) * mult
			} else {
				y = t * mult
			}
			day := mth.SubImage(image.Rect(x-mult, y-mult, x, y)).(*image.Gray)
			days = append(days, day)
			if 0 < n35[0] && n35[0] < n && p <= utils.DayspMonth(ci.start.Month()+time.Month(ii)) {
				rdr, err := os.Open(strconv.Itoa(p) + ".jpg")
				if err != nil {
					return err
				}
				digit, _, err := image.Decode(rdr)
				if err != nil {
					return err
				}
				rdr.Close()
				digitg := utils.Convert(digit, color.GrayModel.Convert)
				digitgg := utils.Copy(digitg)
				today := time.Date(crnt_year, crnt_mnth, p, 12, 0, 0, 0, ci.ven)
				grey, ffst := ci.Color(today)
				if ffst == 0 { // there are no events in moon calendar
					grey = color.Gray{127} // so make everything same shade of gray
				}
				black := color.Gray{0}
				var mark bool
				if (time.Now().Month() == (ci.start.Month()+time.Month(ii)) && time.Now().Day() == p) || (ci.start.Month()+time.Month(ii) == ci.end.Month() && ci.end.Day() == p) {
					mark = true
				}
				for yy := bl.Min.Y; yy < bl.Max.Y; yy++ {
					for xx := bl.Min.X; xx < bl.Max.X; xx++ {
						if gy := digitgg.At(xx, yy).(color.Gray).Y; uint16(gy) > 100 { //!= color.White.Y {
							//digitg.SetGray(xx, yy, grey)
							digitgg.SetGray(xx, yy, grey)
						}
						if mark && (xx < 5 || xx > 45 || yy < 5 || yy > 45) {
							digitgg.SetGray(xx, yy, black)
						}
					}
				}
				draw.Draw(day, day.Bounds(), digitgg, image.ZP, draw.Src)
				p++
			} else {
				draw.Draw(day, day.Bounds(), &image.Uniform{color.Gray{127}}, image.ZP, draw.Src)
				if n == 39 { // mark Wednesday
					draw.Draw(day, day.Bounds(), digitw, image.ZP, draw.Src)

				}
				if n == 42 { // mark month
					grey := color.Gray{127}
					for yy := bl.Min.Y; yy < bl.Max.Y; yy++ {
						for xx := bl.Min.X; xx < bl.Max.X; xx++ {
							if gy := monthnameg.At(xx, yy).(color.Gray).Y; uint16(gy) > 40 { //!= color.White.Y {
								monthnameg.SetGray(xx, yy, grey)
							}
						}
					}
					draw.Draw(day, day.Bounds(), monthnameg, image.ZP, draw.Src)
				}
			}
		}
		//return mth, nil
		//ci.images = append(ci.images, mth)
		ci.images[ii] = mth
	}
	return nil
	//draw.Draw(mth, mth.Bounds(), &image.Uniform{grey}, image.ZP, draw.Src)

}

func (ci *CalImage) Color(day time.Time) (color.Gray, int) {

	for i, ee := range *ci.vnts {
		start, end := parseDate(ee.Start.Date), parseDate(ee.End.Date)
		if start.Before(day) && end.Before(day) {
			continue
		} else if start.Before(day) && end.After(day) { //bingo
			return color.Gray{uint8(shade[ee.Summary])}, i
		} else {
			diff := start.Sub(day).Hours() / 24.0
			grey := uint8(math.Abs(float64(shade[ee.Summary]) - 128.0/diff))

			return color.Gray{grey}, i
		}

	}
	return color.Gray{0}, 0

}

//WriteCloudImage writes the image provided as argument to cloud storage with name provided as argument
func WriteCloudImage(ctx context.Context, mth *image.Gray, filename string) error {

	var err error
	//[START get_default_bucket]
	if bucket == "" {
		if bucket, err = file.DefaultBucketName(ctx); err != nil {
			log.Printf("failed to get default GCS bucket name: %v\n", err)
			return err
		}
	}
	//[END get_default_bucket]
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("failed to create client: %v\n", err)
		return err
	}
	defer client.Close()
	wc := client.Bucket(bucket).Object(filename).NewWriter(ctx)
	wc.ContentType = "image/jpeg"
	wc.ACL = []storage.ACLRule{{storage.AllUsers, storage.RoleReader}}
	if err = jpeg.Encode(wc, mth, nil); err != nil {
		log.Printf("failed to write: %v\n", err)
		return err
	}
	if err = wc.Close(); err != nil {
		log.Printf("failed to close: %v\n", err)
		return err
	}
	log.Printf("updated object: %v\n", wc.Attrs())

	return err

}

//ReadCloudImage reads the jpeg file with filename as argument stored in GCS bucket
func ReadCloudImage(ctx context.Context, filename string) (*image.Image, error) {

	var err error
	//[START get_default_bucket]
	if bucket == "" {
		if bucket, err = file.DefaultBucketName(ctx); err != nil {
			log.Printf("failed to get default GCS bucket name: %v\n", err)
			return nil, err
		}
	}
	//[END get_default_bucket]
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("failed to create client: %v\n", err)
		return nil, err
	}
	defer client.Close()

	rc, err := client.Bucket(bucket).Object(filename).NewReader(ctx)
	if err != nil {
		log.Printf("readFile: unable to open file from bucket %q, file %q: %v", bucket, filename, err)
		return nil, err
	}
	defer rc.Close()

	slurp, err := jpeg.Decode(rc)
	if err != nil {
		log.Printf("readFile: unable to read data from bucket %q, file %q: %v", bucket, filename, err)
		return &slurp, err
	}

	return &slurp, nil
}

//Overflow checks whether any of the months in the period overflows a 7x5 grid. For instance if 31 day month starts on Friday
func (ci *CalImage) Overflow() []int {

	mths := make([]int, 0)
	if ci.OneYear() {
		for n := ci.start.Month(); n <= ci.end.Month(); n++ {
			nd := utils.DayspMonth(n)
			first := int(time.Date(ci.start.Year(), n, 1, 12, 0, 0, 0, ci.ven).Weekday()) + 1 // Weekday starts at Sunday = 0
			if nd+first > 35 {
				mths = append(mths, first-8)
			} else {
				mths = append(mths, 0)
			}
		}
		return mths
	} else {
		return mths
	}

}

//OneYear checks whether start and end are in same year
func (ci *CalImage) OneYear() bool {

	return ci.start.Year() == ci.end.Year()

}

//WriteImage (Depracated) writes the image provided as argument to the file with name provided as argument
func WriteImage(mth *image.Gray, filename string) error {

	f, err := os.Create(filename) //("g127.jpg")
	if err != nil {
		return err
	}
	defer f.Close()
	err = jpeg.Encode(f, mth, nil)
	if err != nil {
		return err
	}
	return nil

}

//WriteImageCloud writes the image provided as argument to cloud storage with name provided as argument
func WriteImageCloud(ctx context.Context, mth *image.Gray, filename string) error {

	var err error
	//[START get_default_bucket]
	if bucket == "" {
		if bucket, err = file.DefaultBucketName(ctx); err != nil {
			log.Printf("failed to get default GCS bucket name: %v\n", err)
			return err
		}
	}
	//[END get_default_bucket]
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("failed to create client: %v\n", err)
		return err
	}
	defer client.Close()
	wc := client.Bucket(bucket).Object(filename).NewWriter(ctx)
	wc.ContentType = "image/jpeg"
	wc.ACL = []storage.ACLRule{{storage.AllUsers, storage.RoleReader}}
	if err = jpeg.Encode(wc, mth, nil); err != nil {
		log.Printf("failed to write: %v\n", err)
		return err
	}
	if err = wc.Close(); err != nil {
		log.Printf("failed to close: %v\n", err)
		return err
	}
	log.Printf("updated object: %v\n", wc.Attrs())

	return err

}
