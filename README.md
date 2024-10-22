# supervent

A synthetic log generator that produces high volumes of realistic log events. Event formats for various sources (an app server, Akamai, etc) are configured in config.json. Behavior of these configured sources will be controlled by defining scenarios (even "normal traffic" is just another scenario) that can be overlaid to simulate real-world patterns, e.g. an outage, Black Friday, a suspicious user, etc. For now, supervent just round-robins through its sources, and adds an extra key-value pair of the form "source: Cisco ASA Firewall" to each event because it's still in testing.

There is a Python version and a Go version. They do the same thing. 

# Roadmap
This is Phase I of the project. For now I am focused on creating realistic events. I am not an expert at event formats. Corrections and additions are welcome, whether submitted as config file entries or sent to me to deal with myself.

Phase II will add behavioral config & control to make each source behave as desired, and to orchestrate patterns across the entire simulated network. Scenarios will be prompts to an LLM that generates the behavioral configurations for the sources with which supervent has been configured. To be 100% clear, all event texts will be generated from scratch by supervent code. There'll be no way for an AI to accidentally leak personal or company-confidential info from its training data into the event stream.

# Disclosure
I am a 50% part-time consultant to Axiom (axiom.co), for whom I write marketing stuff. This is my own free-time project.

Paul Boutin
boutin@gmail.com

# Usage

## Command-Line Arguments

- **--config**: Specifies the path to the configuration file. If not provided, it defaults to config.json.
- **--dataset**: Specifies the Axiom dataset name. This parameter is required.
- **--api_key**: Specifies the Axiom API key. This parameter is required.
- **--batch_size**: Specifies the batch size for HTTP requests. If not provided, it defaults to `100`.
- **--postgres_host**: Specifies the PostgreSQL host. This parameter is optional.
- **--postgres_port**: Specifies the PostgreSQL port. If not provided, it defaults to `5432`.
- **--postgres_db**: Specifies the PostgreSQL database name. This parameter is optional.
- **--postgres_user**: Specifies the PostgreSQL user. This parameter is optional.
- **--postgres_password**: Specifies the PostgreSQL password. This parameter is optional.


- **--config**
  - **Description**: Path to the configuration file.
  - **Type**: String
  - **Default**: config.json
  - **Example**: `--config /path/to/config.json`

- **--dataset**
  - **Description**: Axiom dataset name.
  - **Type**: String
  - **Required**: Yes
  - **Example**: `--dataset supervent`

- **--api_key**
  - **Description**: Axiom API key.
  - **Type**: String
  - **Required**: Yes
  - **Example**: `--api_key xaat-0e268974-2001-4c1f-a747-619dac5257f1`

- **--batch_size**
  - **Description**: Batch size for HTTP requests.
  - **Type**: Integer
  - **Default**: `100`
  - **Example**: `--batch_size 50`

- **--postgres_host**
  - **Description**: PostgreSQL host.
  - **Type**: String
  - **Example**: `--postgres_host localhost`

- **--postgres_port**
  - **Description**: PostgreSQL port.
  - **Type**: Integer
  - **Default**: `5432`
  - **Example**: `--postgres_port 5432`

- **--postgres_db**
  - **Description**: PostgreSQL database name.
  - **Type**: String
  - **Example**: `--postgres_db supervent_db`

- **--postgres_user**
  - **Description**: PostgreSQL user.
  - **Type**: String
  - **Example**: `--postgres_user dbuser`

- **--postgres_password**
  - **Description**: PostgreSQL password.
  - **Type**: String
  - **Example**: `--postgres_password dbpassword`

### Example Usage

**To send Supervent events to an Axiom dataset**

Go version
```sh
./supervent --config /path/to/config.json --dataset supervent --api_key xaat-0e268974-2001-4c1f-a747-619dactt57f1
```
Python version
```sh
python ./supervent.py --config /path/to/config.json --dataset supervent --api_key xaat-0e268974-2001-4c1f-a747-619dactt57f1
```

**To send to a PostgreSQL database**

Go version
```sh
./supervent --config /path/to/config.json  --postgres_host localhost --postgres_port 5432 --postgres_db supervent_db --postgres_user dbuser --postgres_password dbpassword
```
Python version
```sh
python ./supervent.py --config /path/to/config.json  --postgres_host localhost --postgres_port 5432 --postgres_db supervent_db --postgres_user dbuser --postgres_password dbpassword
```

## Source Configuration Parameters 
For config.json or other config file

- **vendor**
  - **Description**: Specifies the vendor or source of the events.
  - **Type**: String
  - **Example Values**: `"F5 Networks BIG-IP"`

- **timestamp_format**
  - **Description**: Specifies the format of the timestamp.
  - **Type**: String
  - **Supported Values**: `UTC`, `ISO`, `Unix`, `RFC3339`
  - **Example Values**: `"UTC"`

- **fields**
  - **Description**: Specifies the fields for the events.
  - **Type**: Object
  - **Example Values**:
    ```json
    {
      "action": {
        "type": "string",
        "allowed_values": ["ALLOW", "DENY"],
        "weights": [0.7, 0.3]
      },
      "response_time": {
        "type": "int",
        "constraints": {
          "min": 1,
          "max": 1000
        },
        "distribution": "normal",
        "mean": 500,
        "stddev": 100
      }
    }
    ```

#### Field Parameters

- **type**
  - **Description**: Specifies the type of the field.
  - **Type**: String
  - **Supported Values**: `string`, `int`, `datetime`
  - **Example Values**: `"string"`

- **allowed_values**
  - **Description**: Specifies the allowed values for the field.
  - **Type**: List of strings
  - **Example Values**: `["ALLOW", "DENY"]`

- **weights**
  - **Description**: Specifies the relative frequency of values in the `allowed_values` list.
  - **Type**: List of floats
  - **Example Values**: `[0.7, 0.3]`

- **constraints**
  - **Description**: Specifies the constraints for the field.
  - **Type**: Object
  - **Example Values**:
    ```json
    {
      "min": 1,
      "max": 1000
    }
    ```

- **distribution**
  - **Description**: Specifies the distribution to use for generating integer values.
  - **Type**: String
  - **Supported Values**: `uniform`, `normal`, `exponential`, `zipfian`, `long_tail`, `random`
  - **Example Values**: `"normal"`

- **mean**
  - **Description**: Specifies the mean value for the `normal` distribution.
  - **Type**: Float
  - **Example Values**: `500`

- **stddev**
  - **Description**: Specifies the standard deviation for the `normal` distribution.
  - **Type**: Float
  - **Example Values**: `100`

- **lambda**
  - **Description**: Specifies the rate parameter for the `exponential` distribution.
  - **Type**: Float
  - **Example Values**: `0.005`

- **s**
  - **Description**: Specifies the parameter for the `zipfian` distribution.
  - **Type**: Float
  - **Example Values**: `1.2`

- **alpha**
  - **Description**: Specifies the parameter for the `long_tail` distribution.
  - **Type**: Float
  - **Example Values**: `2.0`

- **format**
  - **Description**: Specifies the format for the `datetime` field.
  - **Type**: String
  - **Example Values**: `"%Y-%m-%dT%H:%M:%SZ"`

- **formats**
  - **Description**: Specifies the formats for the `message` field.
  - **Type**: List of strings
  - **Example Values**:
    ```json
    [
      "{timestamp} - ERROR - An error occurred while processing the request. Exception: java.lang.NullPointerException",
      "{timestamp} - WARN - Slow response time detected. Response time: 5000ms",
      "{timestamp} - INFO - Application started successfully"
    ]
    ```
