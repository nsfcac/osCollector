package main

import (
	"log"
	"os/exec"
	"fmt"
	"strings"
	//"os"
	"time"
	"strconv"
	"github.com/influxdata/influxdb1-client/v2"
	)

func ExecuteCmd(node string) (map[string]interface{},error){

	out, err := exec.Command("ssh", node, "sensors").Output()
	if err != nil {
		log.Println(err)
		return nil, err
	}
		
	key := ""
	core := ""
	s := strings.Split(string(out),"\n")
	//var m map[string]interface{}
	m := make(map[string]interface{},0)
	for _, line := range s {
	    if (strings.Contains(line,"coretemp")){
	       key = "CPU"
	       wordsList := strings.Split(line,"-")
	       intVal, _ := strconv.Atoi(strings.TrimSpace(wordsList[len(wordsList)-1]))
	        
	       key += strconv.FormatInt(int64(intVal), 10)
	    } else if (strings.Contains(line,":")) {
	    	 words := strings.Split(line,":")
	    	 if (strings.Contains(words[0],"Core")) {
	       	    core =  SpaceFieldsJoin(words[0])
		    strTrimSpace := strings.TrimSpace(words[1])
		    strVal := strings.Trim(strings.Split(strTrimSpace," ")[0],"C +°")
		    val,_ := strconv.ParseFloat(strVal, 64)
		    //val,_ := strconv.ParseInt(strVal, 10, 64)
		    m[key+core] = val
	    	 }
	    }
	}
	m["node"]=node
	return m, nil
}

func SpaceFieldsJoin(str string) string {
    return strings.Join(strings.Fields(str), "")
}

func GetNodeList()[]string{
     nodes := make([]string, 2) 
     nodes = append(nodes,"zc-92-32","zc-92-33")

     for n := 1; n <= 31; n++ {
        postfix := strconv.FormatInt(int64(n), 10)
        nodes = append(nodes,"zc-91-"+postfix,"zc-92-"+postfix)                                                                   
     }  
     return nodes                                                                                                         
}

const (
      MyDB = "monitoring_HW_Sch_OS"
      username = ""
      password = ""
)
func main() {
   
   //node := os.Args[1]
   
   nodes := GetNodeList()  

   // InfluxDB initialization
   // Create a new HTTPClient
   c, err := client.NewHTTPClient(client.HTTPConfig{
      Addr:     "http://10.101.92.201:8086",
      Username: "",
      Password: "",
   })
   if err != nil {
      log.Println(err)
   }
   defer c.Close()

   // Create a new point batch
   bp, err := client.NewBatchPoints(client.BatchPointsConfig{
       Database:  MyDB,
       Precision: "s",
   })
   if err != nil {
      log.Println(err)
   }

   respc, errc := make(chan map[string]interface{},0), make(chan error)
   ticker := time.NewTicker(60 * time.Second)

   // main loop
   for {
       select {
       	      case <-ticker.C:
	      data := make([]map[string]interface{}, 0)
   	      errorList := make([]string,0)
	      for _, node := range nodes {
	      	  go func(nodeAddress string) {
		     resp, err := ExecuteCmd(nodeAddress)
		     	   if err != nil {
			      errc <- err
	   		      return
			   }
     			   respc <- resp
     		   }(node)															    
   	      }


	      for i := 0; i < len(nodes); i++ {
       	      	  select {
   	      	  	 case res := <-respc:
   	       	       	      data = append(data, res)
			       // Create a point and add to batch
			       host, _ := res["node"].(string)
        		       tags := map[string]string{"NodeId": host,"Label":"CPUThermal"}
        		       fields := res
			       	     
        			pt, err := client.NewPoint("OS", tags, fields, time.Now())

        			if err != nil {
           			   log.Println(err)
        			}
        			bp.AddPoint(pt)
	   
			 case e := <-errc:
	       	       	      errorList = append(errorList, e.Error())
   		 }
   	     }
   	   // Write the batch
           if err := c.Write(bp); err != nil {
              log.Println(err)                                                                                                                    } 
	   fmt.Printf("\n DATA: ",data)
           fmt.Printf("\n Total Requests: %d", len(nodes))
           fmt.Printf("\n Total Errors: %d", len(errorList))
       }		  
   }
}
