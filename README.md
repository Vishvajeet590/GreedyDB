# Greedy DB

A simple in-memory key-value datastore written in GO, that could be used for cache. It supports Redis-like features.

- SET (with expiry and exist, not exist strategy)
- GET
- QPUSH
- QPOP
- BQPOP

## Usage


#### Using the hosted service at:
The service is currently active, you can try it live on
[https://greedydb.onrender.com/execute](https://greedydb.onrender.com/execute) with body as
```json
{
	"command": "SET key_a Hello"
}
```
#### Run using Docker Image
```bash
docker build -t greedy .
docker run -p 8080:8080 greedy
```

#### Or you can compile it and run
```bash
go build -o main
./main
```


### Handling key Expiry
Another interesting task was how we handle the key expiry, we could have just done a simple check each time a GET query is made on the key and deleted if the time has elapsed. But it leads to a problem.

Suppose we SET 10 million keys and don’t access them, these 10 mil records are stored inside our memory forever even though the time of expiry has lapsed.

To solve this we can have an Active Key Deletion method which is very easy to implement. We can have a priority key with the expiry time as a priority, so at any given moment we will have the key which has to expire in the coming time before any other key in datastore.

![image](https://github.com/Vishvajeet590/GreedyDB/assets/42716731/5a136d34-897c-45cb-a360-d155acdc8f23)


Now we run 

### Parser
I took this as an opportunity to write a small query parser. It is a very simple LL1 parser, which is left to right predictive parser. LL1 parser has two phases 
- **Tokenizer -** It extracts every word i.e “tokens" ****from the query into an array 
example: “SET key_a Hello EX 10” It will be tokenized  []strings{”SET”, “key_a”, “Hello”, “EX”, “10”}
- **Syntactic Analysis-** In this phase, the parser looks at the tokens starting from the 0th index i.e. currently “SET” and decides what is the next possible it should get.

Now let's look at what we will get at the end of the parser, It is a ParsedQuery struct that holds all the possible values within the valid query scope. After parsing the query we will get this struct which we will use in further execution.

```go
type ParsedQuery struct {
	Type              string
	Key               string
	Value             string
	Expiry            bool
	ExpiryTime        time.Duration
	KeyExistCondition int
	ListName          string
	ListValues        []string
}
```

### How this parser work

Best way to explain it is using FSM, Finite State Machine. 

![image](https://github.com/Vishvajeet590/GreedyDB/assets/42716731/f4b7d5be-9f41-47bc-9b83-1c8fe89808be)


At a given time there are only a few tokens in a correct sequence that are valid, so upon finding these tokens, we advance ahead. The above image shows all the possible sequences for the **SET** query.

Lets take the example of this Tokenized command, []strings{”SET”, “key_a”, “Hello”, “EX”, “10”}
So, on start state parser checks what type of query it is “SET” “GET” etc, on recognizing it parser predicts the next token by setting up the **********Step********** field.
The next only possible token is key_name hence the **step = stepSetKeyName**,  if the parser doesn't encounter key_name it on transitions throws an error.

For the **SET** query the state table can be defined as

| Current Step | Next Step |
| --- | --- |
| SET | key_name |
| key_name | value |
| value | EX or NX/XX or Final |
| EX | NX/XX of Final |
| NX/XX | Final |

The above state table can be implemented with a Switch case inside a for loop, looping over Tokens slice. 

```go
for {
		if p.i > len(p.queryTokens) {
			return p.Query, nil
		}
		switch p.step {
			case stepType:
						switch strings.ToUpper(p.peek()) {
						case "SET":
							p.Query.Type = "SET"
							p.step = stepSetKeyName
							p.pop()
						case "GET":
							p.Query.Type = "GET"
							p.step = stepGetKeyName
							p.pop()
						default:
							return nil, fmt.Errorf("invalid command")
			
			case stepSetKeyName:
					//implementation
					p.step = stepValue
					p.pop()
		
				case stepValue:
					//implementation
					//Now we have a choice if we encounter EX or NX/XX token we transition to respective state else 
					// if no new token is present that means transition to final.
					if p.peek() != "" && p.peek() == "EX" {
						p.step = stepExpiry
					} else if p.peek() != "" && (p.peek() == "NX" || p.peek() == "XX") {
						p.step = stepExist
						continue
					} else {
						return nil, fmt.Errorf("invalid format")
					}
					p.pop()
}
```

You will notice two functions **Peek()** and **********Pop()********** getting used extensively. As LL1 is a protective parser, while it is in the state it looks i.e. peek in the tokens slice for the next coming token and makes the decision according to if the next token is EX then we move to step **stepExpiry** by making **step = stepSetKeyName** which is checked by switch case.

a separate go routine which will peek at the top of PQ and check if time.Now() > exp (exp is epoch time at which key will expire). when this condition is true we use the key in item to delete it from map and then item is popped out of heap. 
By this way we ensure that we have deleted the expired key at the exact moment they expire.


## Commands 

### SET 
Write the value to the store, it has a few optional parameters

- **EX** - to set the key expiry time limit

```json
{
	"command": "SET key_a Hello EX 10"
}
```

- **NX/XX** - Specifies the decision to take if the key already exists. 
NX -- Only set the key if it does not already exist.
XX -- Only set the key if it already exists.

```json
{
	"command": "SET key_a Hello NX"
}
{
	"command": "SET key_a Hello XX"
}
```
### GET
Retrieve the key value using GET
```json
{
	"command": "GET key_a"
}
```
### QPUSH 
The data store can also store a list of values using QPUSH, there can be multiple lists at a given time. We can also update the same list by doing QPUSH again with same list name. 
```json
{
	"command": "QPUSH list_A 1 2 3 4"
}
```

### QPOP
To pop the last stored value of the list.
```json
{
	"command": "QPOP list_A"
}
```
### BQPOP 
This command is very similar to QPOP but, if the list is empty it waits for X seconds, and if with X seconds some other client pushes Valuse into the same queue it returns that.
```json
{
	"command": "BQPOP list_A 10"
}
```
Here if list_A is empty datastore waits for 10 secs, waiting if somone pushes the value into the queue. Then it returns that newly pushed value.

