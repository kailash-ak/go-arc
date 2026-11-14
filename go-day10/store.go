package main
import "sync"
type URL struct{
	Code string `json:"code"`
	LongUrl string `json:"longUrl"`
}

type Store struct{
	mu sync.RWMutex
	urls map[string]string
}

func NewStore() *Store{
	return &Store{
		urls:make(map[string]string),
	}
}
func (s *Store) Get(code string) (string,bool){
	s.mu.RLock()
	defer s.mu.RUnlock()
	url,ok:=s.urls[code]
	//here ok represents whether the code exists in the map or not. If it does, we return the corresponding long URL and true.
	if !ok{
		return "", false
	}
	return url, ok
}

func (s *Store) Delete(code string) bool{
	s.mu.Lock()
	defer s.mu.Unlock()
	_,ok:=s.urls[code]
	if !ok{
		return false
	}
	delete(s.urls,code)
	return true
}

func (s *Store) List() []URL{
	s.mu.RLock()
	defer s.mu.RUnlock()
	urls:=make([]URL,0,len(s.urls))
	for code, longurl:=range s.urls{
		urls = append(urls, URL{Code:code, LongUrl:longurl})
	}
	return urls
}
func(s *Store)Create(longUrl string) (string,error){
	for{
		shortCode, err:=generateCode()
	if err!=nil{
		return "", err
	}
	s.mu.Lock()
	
	if _,ok:=s.urls[shortCode];ok{
		s.mu.Unlock()
		continue
	}
	s.urls[shortCode] = longUrl
	s.mu.Unlock()
	return shortCode, nil
	}
	
}