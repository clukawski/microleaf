# Microleaf

microleaf is a small CLI tool for controlling Nanoleaf. Based off of [`picoleaf`](https://github.com/tessro/picoleaf), but with multi-panel support added (and some os-specific things removed).

## Installation

### Source

Make sure Go is installed, and that `$GOPATH/bin` is on your `$PATH`. Then run:

```bash
go install github.com/clukawski/microleaf
```

# Getting Started

Picoleaf expects a `.microleafrc` file (`toml` formatted) in your home directory, with the following settings (including a [[host_configs]] section for each panel you panel you wish to manage), for example:

```toml
[[host_configs]]
panel_name="outhouse"
host="192.168.1.69:16021"
access_token="8fJ2qP0xL7mN4rT1cV6bH9aG3dQwE5uI"

[[host_configs]]
panel_name="dungeon"
host="192.168.1.69:16021"
access_token="ZsYxWvUtrqPnMmLkJiHhGgFfEeDdCcBb"
```

You can find your Nanoleaf's IP address via your router console. [The Nanoleaf rest API's port is `16021`](https://www.postman.com/postman/postman-team-collections/documentation/5xpm63x/nanoleaf?entity=request-95e89b6d-7272-49cf-907c-bbbebe2c136a).

To create an access token, you'll need to do the following:

1. On your Nanoleaf controller, hold the on-off button for 5-7 seconds until the
   LED starts flashing in a pattern.
2. Within 30 seconds, run: `curl -iLX POST http://<ip address>:<port>/api/v1/new`

This should print a token to your console.

## Usage

```bash
# Power
microleaf -n <panel_name> on   # Turn Nanoleaf on
microleaf -n <panel_name> off  # Turn Nanoleaf off

# Colors
microleaf -n <panel_name> hsl <hue> <saturation> <lightness>  # Set Nanoleaf to the provided HSL
microleaf -n <panel_name> rgb <red> <green> <blue>            # Set Nanoleaf to the provided RGB
microleaf -n <panel_name> temp <temperature>                  # Set Nanoleaf to the provided color temperature
microleaf -n <panel_name> brightness <temperature>            # Set Nanoleaf to the provided brightness

# Effects
microleaf -n <panel_name> effect list           # List installed effects
microleaf -n <panel_name> effect select <name>  # Activate the named effect
microleaf -n <panel_name> effect custom [<panel> <red> <green> <blue> <transition time>] ...

# Panel properties
microleaf -n <panel_name> panel info     # Print all panel information
microleaf -n <panel_name> panel model    # Print Nanoleaf model
microleaf -n <panel_name> panel name     # Print Nanoleaf name
microleaf -n <panel_name> panel version  # Print Nanoleaf and rhythm module versions
```
