+++
date = "2020-03-27T08:00:00+00:00"
title = "Matt's remote workspace in 2020"
subtitle = "A description of the audio/video/computer equipment that Matt uses working from home."
+++

With a lot of folks working remotely in recent days, [I posted a tweet](https://twitter.com/mdlayher/status/1242867571407368192)
about how I decided to tidy up my desk, and I received a lot of questions and
comments about my remote workspace setup! I've been working from home for
about 6 years now and have accumulated a fair amount of gear over time. This
post will detail the equipment I use as part of my workspace, and describe
some of my experiences working with this equipment on a daily basis.

If you're only interested in a particular section, feel free to skip ahead.
Enjoy!

- [The Diagram](#the-diagram)
- [Computer](#computer)
- [Peripherals](#peripherals)
- [Audio](#audio)
- [Compare and contrast](#compare-and-contrast)
- [Conclusion](#conclusion)

![Matt's remote workspace in 2020](/img/blog/matts-remote-workspace-in-2020/desk.jpg)
*Matt's remote workspace in 2020*

## The Diagram

After answering a few questions on Twitter (and for my own reference), I decided
to create a diagram of how each component in my setup is connected. The result
is more complex than I had expected, but here's a legend of cable types mapped
to [Graphviz arrow shapes](https://www.graphviz.org/doc/info/arrows.html):

![Graphviz arrows legend](/img/blog/matts-remote-workspace-in-2020/graphvizarrows.png)
*https://www.graphviz.org/doc/info/arrows.html*

- DisplayPort: box
- HDMI: dot
- RCA: curve
- TRS: inv
- speaker: tee
- TOSLINK/optical: diamond
- USB: normal
- XLR: crow
- other: none

Some devices are connected via more than one cable, such as USB and DisplayPort
for my monitors. In addition, some devices have bidirectional connectivity to
indicate the flow of data in both directions. Behold, The Diagram
(and [source code](https://github.com/mdlayher/mdlayher.com/blob/master/static/misc/workspace.dot))!

[![Workspace setup diagram](/img/blog/matts-remote-workspace-in-2020/workspace.svg)](/img/blog/matts-remote-workspace-in-2020/workspace.svg)

Now that we've seen an overview of how everything fits together, let's break it
down and discuss the advantages and disadvantages of some of my choices.

To be clear, a lot of this equipment is _not at all_ necessary for a typical remote
work setup. A lot of folks can get by just fine with a laptop and maybe an
external display, mouse, or keyboard. But I'm a desktop PC person and "overkill"
is more the way I roll, so let's dive in!

## Computer

The heart of my workspace is my Linux desktop PC. I typically build a new PC
every 4-5 years and this machine was assembled in August 2019.

I decided early-on to go all-in on AMD thanks to their recent return to
competitive status and the success of the Zen and Zen 2 families of CPUs. On
top of that, they're Linux-friendly and actively contribute support for
their CPUs and GPUs to the kernel.

- [CPU: AMD Ryzen 9 3900x](https://www.amazon.com/AMD-Ryzen-3900X-24-Thread-Processor/dp/B07SXMZLP9/)
  - Excellent CPU. I would have loved to wait for the Ryzen 9 3950x, but
  I already had to wait a month or so after the 3900x launched to find one in
  stock.
- [GPU: AMD Radeon Pro WX5100](https://www.newegg.com/amd-100-505940/p/N82E16814105068)
  - I really wanted a full-height AMD workstation GPU (I run Linux and do
  minimal PC gaming) with 3-4 full-size DisplayPort outputs.
  - I probably overspent on the GPU for what I need, and could have went with
  a cheaper half-height card.
- [RAM: Corsair Dominator Platinum 64GiB (4x16GiB) 3200MHz DDR4](https://www.amazon.com/gp/product/B01BGZEQNY/)
  - 3200-3600MHz seems to be the sweet spot for most Zen CPUs. I'm running at
  3200MHz but could consider overclocking if I felt the need.
  - The DIMMs are physically tall: see my notes on CPU cooler below.
- [SSD: Corsair Force MP600 1TB](https://www.amazon.com/gp/product/B01BGZEQNY/)
  - The Corsair MP600 has a large heatsink and to make it fit on my motherboard,
  I had to remove a plastic cover over the chipset fan. This shouldn't be a
  problem, but it's something to note.
- [Motherboard: ASUS ROG Strix X570-E](https://www.amazon.com/gp/product/B07SW8DQVL/)
  - It works, but the hardware sensors don't seem to work in Linux.
  - I am generally a Gigabyte motherboard fan, but there weren't many X570
  options available when I built this machine.
- [PSU: Corsair RM750x](https://www.amazon.com/gp/product/B079HGN5QS/)
  - Modular power supplies are a must and Corsair makes some of the best.
- [CPU cooler: Noctua NH-U14S](https://www.amazon.com/gp/product/B00C9FLSLY)
  - Noctua coolers and fans are my preferred choice for a quiet and cool PC.
  - I almost certainly won't do any overclocking, but I should have headroom
  to do so if I change my mind.
  - I had to move the CPU cooler fan to the exhaust side of the heatsink to
  make enough clearance for my tall RAM.
- [Case: Fractal Design Define R6 USB-C](https://www.newegg.com/black-fractal-design-define-r6-atx-mid-tower/p/N82E16811352089)
  - I can't stand the RGB lighting trend and love the minimal and clean look of
  Fractal Design cases.
  - The case is larger than I need since I don't have a massive GPU or any HDDs
  in this system (I have a custom NAS), but it's easily the best case I've ever
  worked with.

All in all, the machine cost about $2300, which seems reasonable for a machine
that will last 4-5 years. I think I struck a decent balance between
price/performance without going all-out on enthusiast gear like Threadripper
CPUs, RAID SSDs, 128+GiB RAM, etc.

![Completed PC build](/img/blog/matts-remote-workspace-in-2020/pc.jpg)
*Completed PC build with plenty of space for activities*

As for software, I'm currently running a mostly-stock Ubuntu 19.10. I typically
only run LTS releases, but wanted to play with better display scaling options
in newer GNOME. I ended up being able to make things work for me by using a 1.5x
font scaling option in `gnome-tweaks`. I've had some weird GNOME instability
though and am looking forward to being back on LTS with 20.04.

Because I'm running an AMD CPU and GPU, I'm able to get a decent amount of
hardware sensor information within Linux. I am temporarily using the
[`zenpower`](https://github.com/ocerman/zenpower) kernel module rather than the
upstream `k10temp` because it provides much more granular information for AMD Zen
CPUs, although I understand [improvements are coming in Linux 5.6](https://www.phoronix.com/scan.php?page=news_item&px=AMD-Ryzen-k10temp-CCD-V-Current)
which will allow me to switch back. As for my motherboard, I can see more sensor
data in the UEFI, but the hardware sensors don't seem to work in Linux at
this time.

```
$ sensors
asus-isa-0000
Adapter: ISA adapter
cpu_fan:        0 RPM

amdgpu-pci-0900
Adapter: PCI adapter
vddgfx:       +0.90 V
fan1:        1225 RPM  (min =    0 RPM, max = 5500 RPM)
edge:         +79.0°C  (crit = +99.0°C, hyst = -273.1°C)
power1:       31.16 W  (cap =  47.00 W)

zenpower-pci-00c3
Adapter: PCI adapter
SVI2_Core:    +0.96 V
SVI2_SoC:     +1.08 V
Tdie:         +52.6°C  (high = +95.0°C)
Tctl:         +52.6°C
Tccd1:        +40.2°C
Tccd2:        +40.5°C
SVI2_P_Core:   4.41 W
SVI2_P_SoC:   15.28 W
SVI2_C_Core:  +4.61 A
SVI2_C_SoC:  +14.13 A

```

![screenfetch output](/img/blog/matts-remote-workspace-in-2020/screenfetch.png)
*Output from the `screenfetch` utility*

## Peripherals

Now that we've discussed the PC itself, let's talk about some of the devices
I have attached to it! I previously discussed some of my ergonomic setup in
my blog, [_A programmer's journey with RSI_](/blog/a-programmers-journey-with-rsi/).

- [Desk: Uplift (v1) 72x30" (~182x76cm) standing desk](https://www.upliftdesk.com/adjustable-height-desks/)
  - Not quite a PC peripheral, but it's a motorized standing desk with a
  spacious keyboard tray, a long warranty, and lots of bells and whistles.
  - Expensive, but hugely beneficial from an ergonomic perspective.
- [Keyboard: Kinesis Advantage 2](https://kinesis-ergo.com/keyboards/advantage2-keyboard/)
  - Ergonomic, split, mechanical keyboard. It took me a solid month to adjust
  to typing on it, but I will never go back to a regular keyboard.
  - The Kinesis isn't exactly portable, so I would love to try an
  [Ergodox](https://ergodox-ez.com/) at some point for travel.
- [Trackball: Logitech MX Ergo Plus](https://www.logitech.com/en-us/product/mx-ergo-wireless-trackball-mouse)
  - Ergonomic trackball with a stand that allows it to be tilted up to
  30 degrees.
  - Works great for daily computer use, and surprisingly well for casual games
  like Minecraft.
  - I'm also interested in trying out the [Logitech MX Master 3](https://www.amazon.com/Logitech-Master-Advanced-Wireless-Mouse/dp/B07S395RWD)
  just to change things up.
- [Monitors: 3x Dell UltraSharp U2718Q](https://www.dell.com/si/business/p/dell-u2718q-monitor/pd)
  - Large (27"/69cm), beautiful, 4K displays. I have the leftmost one oriented
  vertically so I can display more text.
      - left (vertical): active work: Linux terminals, `tmux`, CLI applications
      - center: active work: mostly Visual Studio Code and/or web browsing
      - right: passive applications: mostly Slack, music players, or a Twitch stream
  - Each has a built-in USB3 hub, and I use two of the hubs to avoid extra
  cable runs for USB devices.
  - I've considered buying a fourth and my GPU would support it, but I think
  three displays is probably optimal for me.
- [MFA device: Yubico YubiKey 5](https://www.amazon.com/gp/product/B07HBD71HL)
  - In my opinion, a must-have [MFA/2FA](https://en.wikipedia.org/wiki/Multi-factor_authentication)
  device.
- [Webcam: Logitech C922x](https://www.logitech.com/en-us/product/c922-pro-stream-webcam)
  - A remote work essential! A good webcam is a good investment.
  - Basically the C922, but I think it has some streaming/gaming feature on
  Windows.
  - Cheaper than the C922 when I bought it.
- [UPS: APC BR1500G](https://www.amazon.com/gp/product/B003Y24DEU)
  - A UPS is a must for any stationary machine (and for network gear).
  - I've been buying APC gear for 10 years, and it is really satisfying to hear
  the three units I own kick on at the same time when the power dips or cuts out.
  - It's nice that you can run coax cable and Ethernet through it for some added
  electrical protection. This particular one supports 1Gb Ethernet just fine.

Overall, I'm extremely happy with the peripherals I've assembled. Things work
pretty seamlessly and I'm able to work comfortably in both sitting and standing
positions, thanks to the Uplift desk and its spacious desk surface. I have
considered buying VESA monitor arms a few times, but it's not strictly necessary
since the desk surface is large enough as-is.

If you work in technology and have never tried a vertical monitor, I highly
recommend giving it a try! I used to use it almost exclusively for programming
in `vim` before I switched to VSCode a few years ago.

Aside from my desk, my most vital peripheral is definitely the Kinesis
Advantage 2 keyboard. It takes commitment to get used to it, but after about 2.5
years with it, I can't imagine myself using any other keyboard as my daily driver.

![Kinesis Advantage 2 QD](/img/blog/matts-remote-workspace-in-2020/kinesis.jpg)
*The Kinesis Advantage 2 QD. Maybe I'll try out Dvorak one of these days.*

## Audio

Finally, let's talk audio! I started my [audiophile](https://en.wikipedia.org/wiki/Audiophile)
journey and have been accumulating audio gear for the past 9 years or so. It
started with a USB DAC and a set of headphones and snowballed from there.
My most recent additions are a nice microphone and audio interface in March 2020.

Let's start with audio input:

- [Audio interface: Focusrite Scarlett 2i2 (3rd gen)](https://www.amazon.com/gp/product/B07QR73T66)
  - Focusrite doesn't officially support Linux, but it was plug-and-play and
  works nicely.
  - It supports two phantom power/48V-capable XLR/TRS combo inputs for
  microphones and instruments.
  - The interface exposes a USB mass storage device when you first plug it in and
  has a README explaining how to install the Windows/Mac Focusrite Control
  software. You can bypass the software and enable full functionality by holding
  the 48V button for 5 seconds, but this is only explained on page 9 of the manual.
- [Microphone: Audio-Technica AT2035](https://www.amazon.com/gp/product/B00D6RMFG6)
  - It's a cardioid pickup pattern condenser microphone, which means:
      - It has reduced sensitivity on the sides and rear, so there's less background
      noise in your audio.
      - It requires phantom power, which I use the Scarlett 2i2 to provide.
  - ~$150 didn't seem unreasonable for a decent professional microphone and I
  suspect I won't need to upgrade for a long time, if ever.
- Accessories: [Monoprice XLR cable](https://www.amazon.com/gp/product/B001URFZKM), [a desk-mount microphone arm](https://www.amazon.com/gp/product/B07V2FJL54), [and a pop filter](https://www.amazon.com/gp/product)
  - Use whatever works for you. I've always liked Monoprice cables, but for the
  mic arm and pop filter, I don't have enough experience yet to provide solid
  recommendations.
  - I do wonder if I should have gotten a longer mic arm for more flexibility.

To be clear, this kind of setup is not necessary for your average video call. I
decided to make the investment because I work remotely full-time anyway and I
wanted the additional audio input flexibility. I play bass guitar from time to
time, and I'm also very interested in doing some [live-coding streams on Twitch](https://www.twitch.tv/mdlayher).
More to come on that soon hopefully!

![Sennheiser HD650 headphones, Audio-Technica AT2035 mic, Onkyo TX-8255 receiver](/img/blog/matts-remote-workspace-in-2020/receiver.jpg)
*Left to right: Sennheiser HD650 headphones, Audio-Technica AT2035 mic, Onkyo TX-8255 receiver. Bonus cameo by Gopher Bobby and some `cmatrix` fun.*

Next, on to audio output:

- [USB DAC: Schiit Modi 2 Uber](https://schiit.com/public/upload/PDF/modi_2_uber_manual.pdf)
  - Superseded by the newer Modi 3.
  - Supports USB, TOSLINK/optical, and coaxial audio inputs. I'm only using USB
  at the moment, but have plans to plug in my Nintendo Switch via TOSLINK.
  - RCA output to my stereo receiver.
- [Headphone amp: Schiit Vali 2](https://www.schiit.com/products/vali-1)
  - A miniature tube hybrid headphone amplifier, used to drive my Sennheiser
  HD650s.
  - Attached to the tape-out RCA jacks from my stereo receiver, so I can
  use any receiver source with the amp.
- [Headphones: Sennheiser HD650](https://www.amazon.com/Sennheiser-HD-650-Professional-Headphone/dp/B00018MSNI/)
  - I've had these for 9 years and can confidently say they are one of the best
  sets of headphones ever made. If you're serious about audio and can use
  open-back headphones (I cannot recommend them in a typical open-office
  plan), buy these.
      - [Drop (formerly Massdrop) often has the HD6xx available](https://drop.com/buy/massdrop-sennheiser-hd6xx).
      I have never tried these but have heard good things.
  - Probably the last reasonable step before you get into really, really
  expensive headphones and audio gear.
- [Stereo receiver: Onkyo TX-8255](https://www.intl.onkyo.com/downloads/manuals/pdf/tx-8255_manual_e.pdf)
  - Discontinued, but it's been flawless for the last 9 years.
  - Used primarily with my Schiit Modi 2, but I occasionally use 3.5mm auxiliary
  input and FM radio as well.
  - I've considered replacing it with an HDMI-capable receiver so I can have
  more video devices on my desk, but that'd be a stretch even for me.
- [Speakers: Polk Audio Monitor 40 Series II](https://www.amazon.com/Polk-Audio-Monitor-Bookshelf-Speaker/dp/B0071MSYEE)
  - My starter speakers from 9 years ago that I have yet to upgrade!
  - When I have more space, I'd consider replacing these and the subwoofer with
  a set of proper full-range, floor-standing speakers.
- [Subwoofer: Polk Audio PSW505](https://www.amazon.com/Polk-Audio-PSW505-Powered-Subwoofer/dp/B000092TT0)
  - My starter subwoofer as well, purchased after I realized I wanted a little
  more kick from the Monitor 40s.
  - The volume is turned way down because I live in an apartment and don't want
  all my neighbors to hate me, but it definitely gets the job done.
  - Attached to speaker outputs from my receiver. The built-in crossover is used
  to pass higher frequencies on to my speakers.

For video calls (or if I'm hacking on something later in the evening) I turn on
the Vali 2 and use my Sennheiser HD650s along with the new microphone rig.
Otherwise, I generally use my speaker setup since I'm home alone for the
majority of the day.

![Focusrite Scarlett 2i2 (3rd gen) interface, Schiit Modi 2 Uber DAC, Schiit Vali 2 headphone amp](/img/blog/matts-remote-workspace-in-2020/audiostack.jpg)
*Top to bottom: Focusrite Scarlett 2i2 (3rd gen) interface, Schiit Modi 2 Uber DAC, Schiit Vali 2 headphone amp. Gopher figurines courtesy of GopherCon.*

![Polk Audio Monitor 40 Series II speaker and Polk Audio PSW505 subwoofer](/img/blog/matts-remote-workspace-in-2020/speakers.jpg)
*Polk Audio Monitor 40 Series II speaker and Polk Audio PSW505 subwoofer. An old laptop box is used as a stand because I haven't found anything better. Suggestions welcome!*

## Compare and contrast

I have been working from home for almost 6 years now and have put a lot of time
and thought into fine-tuning my setup. I've talked to quite a few people about
their remote work setups and I'm often surprised by just how different our
preferences are.

For example, check out my friend [Fatih Arslan's beautiful, minimal desk setup](https://twitter.com/fatih/status/1241177394980868096):

![Fatih's minimal desk setup](/img/blog/matts-remote-workspace-in-2020/fatihdesk.jpg)
*Fatih's minimal desk setup*

Fatih uses a MacBook Pro with an LG 4K monitor and an Apple keyboard and trackpad.
This setup is objectively beautiful and free of unnecessary clutter and
distractions. It works extremely well for Fatih and his preferred style of
remote work. For me, I think I'd really miss my speakers and two extra displays!

Next, we have my friend [Andrew Herrington](https://twitter.com/andrewthetechie)'s
six(!) monitor workstation:

![Andrew's six monitor workstation](/img/blog/matts-remote-workspace-in-2020/andrewdesk.jpg)
*Andrew's six monitor workstation*

Andrew uses two computers: a MacBook Pro with an external GPU to drive four
displays, and a custom PC to drive the other two, with all of the monitors
mounted on a single arm. He has a Blue Yeti microphone and a couple of nice
keyboards: the Ergodox and Unicomp 103. Andrew makes use of all that screen real
estate with full-screen browsers, videos, source code, and video call windows. I
can't even imagine finding a use for twice as many monitors as I have now, but
this is Andrew's ideal remote work setup!

## Conclusion

Ultimately, it is up to you to decide what kind of setup works best for you.
Some folks even travel the world and work nomadically from various coffee shops
and cafes every day! I hope my friends and I have been able to give you some
inspiration for making your remote workspace truly your own.

For me, my optimal work environment is my home office with a Linux desktop PC,
three monitors, and as many high-quality audio/video peripherals as I can use!

If you have any questions, feel free to contact me! I'm mdlayher on
[Gophers Slack](https://gophers.slack.com/), [GitHub](https://github.com/mdlayher)
and [Twitter](https://twitter.com/mdlayher).

Special thanks to [Fatih Arslan (@fatih)](https://twitter.com/fatih) and
[Andrew Herrington (@andrewthetechie)](https://twitter.com/andrewthetechie) for
allowing me to share their setups with you, and to [Mary Francois (@MaryFrancois)](https://twitter.com/MaryFrancois)
for proofreading.
